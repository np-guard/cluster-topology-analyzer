/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	ocroutev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

// Create a common.Resource object from a k8s Workload object
func ScanK8sWorkloadObject(kind string, objDataBuf []byte) (*common.Resource, error) {
	var podSpecV1 v1.PodTemplateSpec
	var resourceCtx common.Resource
	var metaObj metaV1.Object
	resourceCtx.Resource.Kind = kind
	switch kind { // TODO: handle Pod
	case "ReplicaSet":
		obj := parseResource[appsv1.ReplicaSet](bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.GetLabels()
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	case "ReplicationController":
		obj := parseResource[v1.ReplicationController](bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		podSpecV1 = *obj.Spec.Template
		metaObj = obj
	case "Deployment":
		obj := parseResource[appsv1.Deployment](bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	case "DaemonSet":
		obj := parseResource[appsv1.DaemonSet](bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	case "StatefulSet":
		obj := parseResource[appsv1.StatefulSet](bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	case "Job":
		obj := parseResource[batchv1.Job](bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	default:
		return nil, fmt.Errorf("unsupported object type: `%s`", kind)
	}

	parseDeployResource(&podSpecV1, metaObj, &resourceCtx)
	return &resourceCtx, nil
}

func matchLabelSelectorToStrLabels(labels map[string]string) []string {
	res := []string{}
	for k, v := range labels {
		res = append(res, fmt.Sprintf("%s:%s", k, v))
	}
	return res
}

func ScanK8sConfigmapObject(kind string, objDataBuf []byte) (*common.CfgMap, error) {
	obj := parseResource[v1.ConfigMap](bytes.NewReader(objDataBuf))
	if obj == nil {
		return nil, fmt.Errorf("unable to parse configmap")
	}

	fullName := obj.ObjectMeta.Namespace + "/" + obj.ObjectMeta.Name
	return &common.CfgMap{FullName: fullName, Data: obj.Data}, nil
}

// Create a common.Service object from a k8s Service object
func ScanK8sServiceObject(kind string, objDataBuf []byte) (*common.Service, error) {
	if kind != "Service" {
		return nil, fmt.Errorf("expected parsing a Service resource, but got `%s`", kind)
	}

	svcObj := parseResource[v1.Service](bytes.NewReader(objDataBuf))
	if svcObj == nil {
		return nil, fmt.Errorf("failed to parse Service resource")
	}
	var serviceCtx common.Service
	serviceCtx.Resource.Name = svcObj.GetName()
	serviceCtx.Resource.Namespace = svcObj.Namespace
	serviceCtx.Resource.Kind = kind
	serviceCtx.Resource.Type = svcObj.Spec.Type
	serviceCtx.Resource.Selectors = matchLabelSelectorToStrLabels(svcObj.Spec.Selector)
	serviceCtx.Resource.ExposeExternally = (svcObj.Spec.Type == v1.ServiceTypeLoadBalancer || svcObj.Spec.Type == v1.ServiceTypeNodePort)
	serviceCtx.Resource.ExposeToCluster = false

	for _, p := range svcObj.Spec.Ports {
		n := common.SvcNetworkAttr{Port: int(p.Port), TargetPort: p.TargetPort, Protocol: p.Protocol}
		serviceCtx.Resource.Network = append(serviceCtx.Resource.Network, n)
	}

	return &serviceCtx, nil
}

// Scan an OpenShift Route object and mark the services it uses to be exposed inside the cluster
func ScanOCRouteObject(kind string, objDataBuf []byte, servicesToExpose common.ServicesToExpose) error {
	if kind != "Route" {
		return fmt.Errorf("expected parsing a Route resource, but got `%s`", kind)
	}

	routeObj := parseResource[ocroutev1.Route](bytes.NewReader(objDataBuf))
	if routeObj == nil {
		return fmt.Errorf("failed to parse Route resource")
	}

	exposedServicesInNamespace, ok := servicesToExpose[routeObj.Namespace]
	if !ok {
		servicesToExpose[routeObj.Namespace] = map[string]bool{}
		exposedServicesInNamespace = servicesToExpose[routeObj.Namespace]
	}
	exposedServicesInNamespace[routeObj.Spec.To.Name] = false
	for _, backend := range routeObj.Spec.AlternateBackends {
		exposedServicesInNamespace[backend.Name] = false
	}

	return nil
}

// Scan an Ingress object and mark the services it uses to be exposed inside the cluster
func ScanIngressObject(kind string, objDataBuf []byte, servicesToExpose common.ServicesToExpose) error {
	if kind != "Ingress" {
		return fmt.Errorf("expected parsing a Ingress resource, but got `%s`", kind)
	}

	ingressObj := parseResource[networkv1.Ingress](bytes.NewReader(objDataBuf))
	if ingressObj == nil {
		return fmt.Errorf("failed to parse Ingress resource")
	}

	exposedServicesInNamespace, ok := servicesToExpose[ingressObj.Namespace]
	if !ok {
		servicesToExpose[ingressObj.Namespace] = map[string]bool{}
		exposedServicesInNamespace = servicesToExpose[ingressObj.Namespace]
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

func parseDeployResource(podSpec *v1.PodTemplateSpec, obj metaV1.Object, resourceCtx *common.Resource) {
	resourceCtx.Resource.Name = obj.GetName()
	resourceCtx.Resource.Namespace = obj.GetNamespace()
	resourceCtx.Resource.ServiceAccountName = podSpec.Spec.ServiceAccountName
	for containerIdx := range podSpec.Spec.Containers {
		container := &podSpec.Spec.Containers[containerIdx]
		resourceCtx.Resource.Image.ID = container.Image
		for _, e := range container.Env {
			if e.Value != "" {
				if netAddr, ok := NetworkAddressValue(e.Value); ok {
					resourceCtx.Resource.NetworkAddrs = append(resourceCtx.Resource.NetworkAddrs, netAddr)
				}
			} else if e.ValueFrom != nil && e.ValueFrom.ConfigMapKeyRef != nil {
				keyRef := e.ValueFrom.ConfigMapKeyRef
				if keyRef.Name != "" && keyRef.Key != "" { // just store ref for now - check later if it's a network address
					cfgMapKeyRef := common.CfgMapKeyRef{Name: keyRef.Name, Key: keyRef.Key}
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
		if netAddr, ok := NetworkAddressValue(val); ok {
			networkAddresses = append(networkAddresses, netAddr)
		}
	}
	return networkAddresses
}

// NetworkAddressValue tries to extract a network address from the given string.
// This is a critical step in identifying which service talks to which,
// because it decides if the given string is an evidence for a potentially required connectivity.
// If it succeeds, a "cleaned" network address is returned as a string, together with the value true.
// Otherwise (there does not seem to be a network address in "value"), it returns "" with the value false.
func NetworkAddressValue(value string) (string, bool) {
	host, err := getHostFromURL(value)
	if err != nil {
		return "", false // value cannot be interpreted as a URL
	}

	hostNoPort := host
	colonPos := strings.Index(host, ":")
	if colonPos >= 0 {
		hostNoPort = host[:colonPos]
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
