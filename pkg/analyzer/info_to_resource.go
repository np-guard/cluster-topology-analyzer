/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	ocroutev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/cli-runtime/pkg/resource"
)

// k8sWorkloadObjectFromInfo creates a Resource object from an Info object
func k8sWorkloadObjectFromInfo(info *resource.Info) (*Resource, error) {
	var podSpecV1 *v1.PodTemplateSpec
	var resourceCtx Resource
	var metaObj metaV1.Object
	resourceCtx.Resource.Kind = info.Object.GetObjectKind().GroupVersionKind().Kind
	switch resourceCtx.Resource.Kind {
	case pod:
		obj := parseResourceFromInfo[v1.Pod](info)
		podSpecV1 = &v1.PodTemplateSpec{Spec: obj.Spec, ObjectMeta: obj.ObjectMeta}
		metaObj = obj
	case replicaSet:
		obj := parseResourceFromInfo[appsv1.ReplicaSet](info)
		podSpecV1 = &obj.Spec.Template
		metaObj = obj
	case replicationController:
		obj := parseResourceFromInfo[v1.ReplicationController](info)
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	case deployment:
		obj := parseResourceFromInfo[appsv1.Deployment](info)
		podSpecV1 = &obj.Spec.Template
		metaObj = obj
	case daemonSet:
		obj := parseResourceFromInfo[appsv1.DaemonSet](info)
		podSpecV1 = &obj.Spec.Template
		metaObj = obj
	case statefulSet:
		obj := parseResourceFromInfo[appsv1.StatefulSet](info)
		podSpecV1 = &obj.Spec.Template
		metaObj = obj
	case cronJob:
		obj := parseResourceFromInfo[batchv1.CronJob](info)
		podSpecV1 = &obj.Spec.JobTemplate.Spec.Template
		metaObj = obj
	case job:
		obj := parseResourceFromInfo[batchv1.Job](info)
		podSpecV1 = &obj.Spec.Template
		metaObj = obj
	default:
		return nil, fmt.Errorf("unsupported object type: `%s`", resourceCtx.Resource.Kind)
	}

	parseDeployResource(podSpecV1, metaObj, &resourceCtx)
	return &resourceCtx, nil
}

func matchLabelSelectorToStrLabels(labels map[string]string) []string {
	res := []string{}
	for k, v := range labels {
		res = append(res, fmt.Sprintf("%s:%s", k, v))
	}
	return res
}

// k8sConfigmapFromInfo creates a CfgMap object from a k8s ConfigMap object
func k8sConfigmapFromInfo(info *resource.Info) (*cfgMap, error) {
	obj := parseResourceFromInfo[v1.ConfigMap](info)
	if obj == nil {
		return nil, fmt.Errorf("unable to parse configmap")
	}

	fullName := obj.ObjectMeta.Namespace + "/" + obj.ObjectMeta.Name
	return &cfgMap{FullName: fullName, Data: obj.Data}, nil
}

// k8sServiceFromInfo creates a Service object from a k8s Service object
func k8sServiceFromInfo(info *resource.Info) (*Service, error) {
	svcObj := parseResourceFromInfo[v1.Service](info)
	if svcObj == nil {
		return nil, fmt.Errorf("failed to parse Service resource")
	}
	var serviceCtx Service
	serviceCtx.Resource.Name = svcObj.GetName()
	serviceCtx.Resource.Namespace = svcObj.Namespace
	serviceCtx.Resource.Kind = svcObj.Kind
	serviceCtx.Resource.Type = svcObj.Spec.Type
	serviceCtx.Resource.Selectors = matchLabelSelectorToStrLabels(svcObj.Spec.Selector)
	serviceCtx.Resource.ExposeExternally = (svcObj.Spec.Type == v1.ServiceTypeLoadBalancer || svcObj.Spec.Type == v1.ServiceTypeNodePort)
	serviceCtx.Resource.ExposeToCluster = false

	for _, p := range svcObj.Spec.Ports {
		n := SvcNetworkAttr{Port: int(p.Port), TargetPort: p.TargetPort, Protocol: p.Protocol}
		serviceCtx.Resource.Network = append(serviceCtx.Resource.Network, n)
	}

	return &serviceCtx, nil
}

// ocRouteFromInfo updates servicesToExpose based on an OpenShift Route object
func ocRouteFromInfo(info *resource.Info, toExpose servicesToExpose) error {
	routeObj := parseResourceFromInfo[ocroutev1.Route](info)
	if routeObj == nil {
		return fmt.Errorf("failed to parse Route resource")
	}

	exposedServicesInNamespace, ok := toExpose[routeObj.Namespace]
	if !ok {
		toExpose[routeObj.Namespace] = map[string]bool{}
		exposedServicesInNamespace = toExpose[routeObj.Namespace]
	}
	exposedServicesInNamespace[routeObj.Spec.To.Name] = false
	for _, backend := range routeObj.Spec.AlternateBackends {
		exposedServicesInNamespace[backend.Name] = false
	}

	return nil
}

// k8sIngressFromInfo updates servicesToExpose based on an K8s Ingress object
func k8sIngressFromInfo(info *resource.Info, toExpose servicesToExpose) error {
	ingressObj := parseResourceFromInfo[networkv1.Ingress](info)
	if ingressObj == nil {
		return fmt.Errorf("failed to parse Ingress resource")
	}

	exposedServicesInNamespace, ok := toExpose[ingressObj.Namespace]
	if !ok {
		toExpose[ingressObj.Namespace] = map[string]bool{}
		exposedServicesInNamespace = toExpose[ingressObj.Namespace]
	}

	defaultBackend := ingressObj.Spec.DefaultBackend
	if defaultBackend != nil && defaultBackend.Service != nil {
		exposedServicesInNamespace[defaultBackend.Service.Name] = false
	}

	for ruleIdx := range ingressObj.Spec.Rules {
		rule := &ingressObj.Spec.Rules[ruleIdx]
		if rule.HTTP != nil {
			for pathIdx := range rule.HTTP.Paths {
				svc := rule.HTTP.Paths[pathIdx].Backend.Service
				if svc != nil {
					exposedServicesInNamespace[svc.Name] = false
				}
			}
		}
	}

	return nil
}

func parseDeployResource(podSpec *v1.PodTemplateSpec, obj metaV1.Object, resourceCtx *Resource) {
	resourceCtx.Resource.Name = obj.GetName()
	resourceCtx.Resource.Namespace = obj.GetNamespace()
	resourceCtx.Resource.Labels = podSpec.Labels
	delete(resourceCtx.Resource.Labels, "pod-template-hash") // auto-generated - better not use it in netpols
	resourceCtx.Resource.ServiceAccountName = podSpec.Spec.ServiceAccountName
	for containerIdx := range podSpec.Spec.Containers {
		container := &podSpec.Spec.Containers[containerIdx]
		resourceCtx.Resource.Image.ID = container.Image
		for _, e := range container.Env {
			if e.Value != "" {
				if netAddr, ok := networkAddressFromStr(e.Value); ok {
					resourceCtx.Resource.NetworkAddrs = append(resourceCtx.Resource.NetworkAddrs, netAddr)
				}
			} else if e.ValueFrom != nil && e.ValueFrom.ConfigMapKeyRef != nil {
				keyRef := e.ValueFrom.ConfigMapKeyRef
				if keyRef.Name != "" && keyRef.Key != "" { // just store ref for now - check later if it's a network address
					cfgMapKeyRef := cfgMapKeyRef{Name: keyRef.Name, Key: keyRef.Key}
					resourceCtx.Resource.ConfigMapKeyRefs = append(resourceCtx.Resource.ConfigMapKeyRefs, cfgMapKeyRef)
				}
			}
		}
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil { // just store ref for now - check later if the config map values contain a network address
				resourceCtx.Resource.ConfigMapRefs = append(resourceCtx.Resource.ConfigMapRefs, envFrom.ConfigMapRef.Name)
			}
		}
		resourceCtx.Resource.NetworkAddrs = appendNetworkAddresses(resourceCtx.Resource.NetworkAddrs, container.Args)
		resourceCtx.Resource.NetworkAddrs = appendNetworkAddresses(resourceCtx.Resource.NetworkAddrs, container.Command)
	}
	for volIdx := range podSpec.Spec.Volumes {
		volume := &podSpec.Spec.Volumes[volIdx]
		if volume.ConfigMap != nil {
			resourceCtx.Resource.ConfigMapRefs = append(resourceCtx.Resource.ConfigMapRefs, volume.ConfigMap.Name)
		}
	}
}

func appendNetworkAddresses(networkAddresses, values []string) []string {
	for _, val := range values {
		if netAddr, ok := networkAddressFromStr(val); ok {
			networkAddresses = append(networkAddresses, netAddr)
		}
	}
	return networkAddresses
}

// networkAddressFromStr tries to extract a network address from the given string.
// This is a critical step in identifying which service talks to which,
// because it decides if the given string is an evidence for a potentially required connectivity.
// If it succeeds, a "cleaned" network address is returned as a string, together with the value true.
// Otherwise (there does not seem to be a network address in "value"), it returns "" with the value false.
// As value may be in the form of e.g. "key:val" where "val" holds the network address, we will check several possible suffixes.
func networkAddressFromStr(value string) (string, bool) {
	suffixes := possibleSuffixes(value)
	for _, suffix := range suffixes {
		addr, ok := networkAddressFromSuffix(suffix)
		if ok {
			return addr, ok
		}
	}
	return "", false
}

func networkAddressFromSuffix(value string) (string, bool) {
	host, err := getHostFromURL(value)
	if err != nil {
		return "", false // value cannot be interpreted as a URL
	}

	hostNoPort := host
	colonPos := strings.Index(host, ":")
	if colonPos >= 0 { // host includes port number or port name
		hostNoPort = host[:colonPos]
		port := host[colonPos+1:] // now validate the port
		if len(validation.IsValidPortName(port)) > 0 {
			portInt, _ := strconv.Atoi(port)
			if len(validation.IsValidPortNum(portInt)) > 0 {
				return "", false
			}
		}
	}

	errs := validation.IsDNS1123Subdomain(hostNoPort)
	if len(errs) > 0 {
		return "", false // host part of the URL is not really a network address
	}

	_, err = strconv.Atoi(hostNoPort)
	if err == nil {
		return "", false // we do not accept integers as network addresses
	}
	return host, true
}

// Sometimes the given value includes the network address as its suffix.
// For example, a command-line arg may look like "server-addr=my_server:5000"
// If we are unable to convert "value" to a network address, we may also want to check its suffixes.
// This function returns all suffixes that start with a letter, and have ':', ' ' or '=' just before this initial letter.
func possibleSuffixes(value string) []string {
	res := []string{value}

	var prevRune rune
	for i, r := range value {
		if i > 1 && unicode.IsLetter(r) && (prevRune == ':' || prevRune == '=' || prevRune == ' ') {
			res = append(res, value[i:])
		}
		prevRune = r
	}

	return res
}

// Attempts to parse the given string as a URL, and extract its Host part.
// Returns an error if the string cannot be interpreted as a URL
func getHostFromURL(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	if parsedURL.Host == "" { // URL looks like scheme:opaque[?query][#fragment]
		parsedURL.Fragment = ""
		parsedURL.RawQuery = ""
		return parsedURL.String(), nil
	}

	// URL looks like [scheme:][//[userinfo@]host][/]path[?query][#fragment]
	return parsedURL.Host, nil
}

func parseResourceFromInfo[T interface{}](info *resource.Info) *T {
	obj, ok := info.Object.(*unstructured.Unstructured)
	if !ok {
		return nil
	}

	var rc T
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &rc)
	if err != nil {
		return nil
	}
	return &rc
}
