/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"fmt"
	"slices"

	"k8s.io/cli-runtime/pkg/resource"

	"github.com/np-guard/netpol-analyzer/pkg/manifests/fsscanner"
)

// K8s resources that are relevant for connectivity analysis
const (
	pod                   string = "Pod"
	replicaSet            string = "ReplicaSet"
	replicationController string = "ReplicationController"
	deployment            string = "Deployment"
	statefulSet           string = "StatefulSet"
	daemonSet             string = "DaemonSet"
	job                   string = "Job"
	cronJob               string = "CronJob"
	service               string = "Service"
	configmap             string = "ConfigMap"
	route                 string = "Route"
	ingress               string = "Ingress"
	httpRoute             string = "HTTPRoute"
	grpcRoute             string = "GRPCRoute"
)

var (
	acceptedK8sKinds = []string{pod, replicaSet, replicationController, deployment, daemonSet, statefulSet, job, cronJob,
		service, configmap, route, ingress, httpRoute, grpcRoute}
)

// resourceAccumulator is used to locate all relevant K8s resources in given file-system directories
// and to convert them into the internal structs, used for later processing.
type resourceAccumulator struct {
	logger       Logger
	stopOn1stErr bool

	workloads        []*Resource      // accumulates all workload resources found
	services         []*Service       // accumulates all service resources found
	configmaps       []*cfgMap        // accumulates all ConfigMap resources found
	servicesToExpose servicesToExpose // stores which services should be later exposed
}

func newResourceAccumulator(logger Logger, failFast bool) *resourceAccumulator {
	res := resourceAccumulator{logger: logger, stopOn1stErr: failFast}

	res.servicesToExpose = servicesToExpose{}

	return &res
}

// A convenience function to call parseK8sYaml() on multiple YAML paths
func (ra *resourceAccumulator) parseK8sYamls(yamlPaths []string) []FileProcessingError {
	parseErrors := []FileProcessingError{}
	for _, mfp := range yamlPaths {
		errs := ra.parseK8sYaml(mfp)
		parseErrors = append(parseErrors, errs...)
		if stopProcessing(ra.stopOn1stErr, parseErrors) {
			return parseErrors
		}
	}

	return parseErrors
}

// parseK8sYaml takes the path to a single YAML file and attempts to parse each of its documents into
// one of the relevant k8s resources
func (ra *resourceAccumulator) parseK8sYaml(mfp string) []FileProcessingError {
	infos, errs := fsscanner.GetResourceInfosFromDirPath([]string{mfp}, false, ra.stopOn1stErr)
	parseErrors := []FileProcessingError{}
	for _, err := range errs {
		parseErrors = appendAndLogNewError(parseErrors, failedReadingFile(mfp, err), ra.logger)
		if stopProcessing(ra.stopOn1stErr, parseErrors) {
			return parseErrors
		}
	}

	moreErrors := ra.parseInfos(infos)
	return append(parseErrors, moreErrors...)
}

// A convenience function to call parseInfo() on multiple Info objects
func (ra *resourceAccumulator) parseInfos(infos []*resource.Info) []FileProcessingError {
	parseErrors := []FileProcessingError{}
	for _, info := range infos {
		err := ra.parseInfo(info)
		if err != nil {
			kind := "<unknown>"
			if info != nil && info.Object != nil {
				kind = info.Object.GetObjectKind().GroupVersionKind().Kind
			}
			parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(kind, info.Source, err), ra.logger)
			if stopProcessing(ra.stopOn1stErr, parseErrors) {
				return parseErrors
			}
		}
	}

	return parseErrors
}

// parseInfo takes an Info object, parses it into a K8s resource and puts it into one of the 3 struct slices:
// the workload resource slice, the Service resource slice and the ConfigMaps resource slice
// It also updates the set of services to be exposed when parsing Ingress or OpenShift Routes
func (ra *resourceAccumulator) parseInfo(info *resource.Info) error {
	if info == nil || info.Object == nil {
		return fmt.Errorf("a bad Info object - Object field is Nil")
	}

	kind := info.Object.GetObjectKind().GroupVersionKind().Kind
	if !slices.Contains(acceptedK8sKinds, kind) {
		msg := fmt.Sprintf("skipping object with type: %s", kind)
		resourcePath := info.Source
		if resourcePath != "" {
			msg = fmt.Sprintf("in file: %s, %s", resourcePath, msg)
		}
		ra.logger.Infof(msg)
		return nil
	}

	var err error
	switch kind {
	case service:
		var svc *Service
		svc, err = k8sServiceFromInfo(info)
		if err == nil {
			ra.services = append(ra.services, svc)
		}
	case route:
		err = ocRouteFromInfo(info, ra.servicesToExpose)
	case ingress:
		err = k8sIngressFromInfo(info, ra.servicesToExpose)
	case httpRoute:
		err = gatewayHTTPRouteFromInfo(info, ra.servicesToExpose)
	case grpcRoute:
		err = gatewayGRPCRouteFromInfo(info, ra.servicesToExpose)
	case configmap:
		var cfgmap *cfgMap
		cfgmap, err = k8sConfigmapFromInfo(info)
		if err == nil {
			ra.configmaps = append(ra.configmaps, cfgmap)
		}
	default:
		var wl *Resource
		wl, err = k8sWorkloadObjectFromInfo(info)
		if err == nil {
			ra.workloads = append(ra.workloads, wl)
		}
	}

	return err
}

// inlineConfigMapRefsAsEnvs appends to the Envs of each given resource the ConfigMap values it is referring to
// It should only be called after ALL calls to getRelevantK8sResources successfully returned
func (ra *resourceAccumulator) inlineConfigMapRefsAsEnvs() []FileProcessingError {
	cfgMapsByName := map[string]*cfgMap{}
	for _, cm := range ra.configmaps {
		cfgMapsByName[cm.FullName] = cm
	}

	parseErrors := []FileProcessingError{}
	for _, res := range ra.workloads {
		// inline the envFrom field in PodSpec->containers
		for _, cfgMapRef := range res.Resource.ConfigMapRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapRef
			if cfgMap, ok := cfgMapsByName[configmapFullName]; ok {
				for _, v := range cfgMap.Data {
					if netAddr, ok := networkAddressFromStr(v); ok {
						res.Resource.NetworkAddrs = append(res.Resource.NetworkAddrs, netAddr)
					}
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name), ra.logger)
			}
		}

		// inline PodSpec->container->env->valueFrom->configMapKeyRef
		for _, cfgMapKeyRef := range res.Resource.ConfigMapKeyRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapKeyRef.Name
			cfgMap, ok := cfgMapsByName[configmapFullName]
			if !ok {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name), ra.logger)
				continue
			}
			if val, ok := cfgMap.Data[cfgMapKeyRef.Key]; ok {
				if netAddr, ok := networkAddressFromStr(val); ok {
					res.Resource.NetworkAddrs = append(res.Resource.NetworkAddrs, netAddr)
				}
			} else {
				err := configMapKeyNotFound(cfgMapKeyRef.Name, cfgMapKeyRef.Key, res.Resource.Name)
				parseErrors = appendAndLogNewError(parseErrors, err, ra.logger)
			}
		}
	}
	return parseErrors
}

// exposeServices changes the exposure of services pointed by resources such as Route or Ingress.
// This will ensure that the network policy for their workloads will allow ingress from all the cluster or from the outside internet.
// It should only be called after ALL calls to getRelevantK8sResources successfully returned
func (ra *resourceAccumulator) exposeServices() {
	for _, svc := range ra.services {
		exposedServicesInNamespace, ok := ra.servicesToExpose[svc.Resource.Namespace]
		if !ok {
			continue
		}
		portsToExpose, ok := exposedServicesInNamespace[svc.Resource.Name]
		if !ok {
			continue
		}
		for i := range svc.Resource.Network {
			port := &svc.Resource.Network[i]
			for _, portToExpose := range portsToExpose {
				if port.equals(portToExpose) {
					port.exposeToCluster = true
				}
			}
		}
	}
}
