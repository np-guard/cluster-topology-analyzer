/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"fmt"
	"regexp"

	"k8s.io/cli-runtime/pkg/resource"

	"github.com/np-guard/netpol-analyzer/pkg/netpol/manifests/fsscanner"
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
	cronJob               string = "CronTab"
	service               string = "Service"
	configmap             string = "ConfigMap"
	route                 string = "Route"
	ingress               string = "Ingress"
)

var (
	acceptedK8sTypesRegex = fmt.Sprintf("(^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$)",
		pod, replicaSet, replicationController, deployment, daemonSet, statefulSet, job, cronJob, service, configmap, route, ingress)
	acceptedK8sTypes = regexp.MustCompile(acceptedK8sTypesRegex)
)

// resourceAccumulator is used to locate all relevant K8s resources in given file-system directories
// and to convert them into the internal structs, used for later processing.
type resourceAccumulator struct {
	logger       Logger
	stopOn1stErr bool
	walkFn       WalkFunction // for customizing directory scan

	workloads        []*Resource      // accumulates all workload resources found
	services         []*Service       // accumulates all service resources found
	configmaps       []*cfgMap        // accumulates all ConfigMap resources found
	servicesToExpose servicesToExpose // stores which services should be later exposed
}

func newResourceAccumulator(logger Logger, failFast bool, walkFn WalkFunction) *resourceAccumulator {
	res := resourceAccumulator{logger: logger, stopOn1stErr: failFast, walkFn: walkFn}

	res.servicesToExpose = servicesToExpose{}

	return &res
}

// getRelevantK8sResources is the main function of resourceFinder.
// It scans a given directory using walkFn, looking for all yaml files. It then breaks each yaml into its documents
// and extracts all K8s resources that are relevant for connectivity analysis.
// The resources are stored in the struct, separated to workloads, services and configmaps
func (rf *resourceAccumulator) getRelevantK8sResources(repoDir string) []FileProcessingError {
	mf := manifestFinder{rf.logger, rf.stopOn1stErr, rf.walkFn}
	manifestFiles, fileScanErrors := mf.searchForManifests(repoDir)
	if stopProcessing(rf.stopOn1stErr, fileScanErrors) {
		return fileScanErrors
	}
	if len(manifestFiles) == 0 {
		fileScanErrors = appendAndLogNewError(fileScanErrors, noYamlsFound(), rf.logger)
		return fileScanErrors
	}

	for _, mfp := range manifestFiles {
		errs := rf.parseK8sYaml(mfp)
		fileScanErrors = append(fileScanErrors, errs...)
		if stopProcessing(rf.stopOn1stErr, fileScanErrors) {
			return fileScanErrors
		}
	}

	return fileScanErrors
}

// parseK8sYaml takes a YAML file and attempts to parse each of its documents into
// one of the relevant k8s resources
func (rf *resourceAccumulator) parseK8sYaml(mfp string) []FileProcessingError {
	infos, errs := fsscanner.GetResourceInfosFromDirPath([]string{mfp}, false, rf.stopOn1stErr)
	parseErrors := []FileProcessingError{}
	for _, err := range errs {
		parseErrors = appendAndLogNewError(parseErrors, failedReadingFile(mfp, err), rf.logger)
		if stopProcessing(rf.stopOn1stErr, parseErrors) {
			return parseErrors
		}
	}

	moreErrors := rf.parseInfos(infos)
	return append(parseErrors, moreErrors...)

}

func (rf *resourceAccumulator) parseInfos(infos []*resource.Info) []FileProcessingError {
	parseErrors := []FileProcessingError{}
	for _, info := range infos {
		err := rf.parseInfo(info)
		if err != nil {
			kind := "<unknown>"
			if info != nil && info.Object != nil {
				kind = info.Object.GetObjectKind().GroupVersionKind().Kind
			}
			parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(kind, info.Source, err), rf.logger)
			if stopProcessing(rf.stopOn1stErr, parseErrors) {
				return parseErrors
			}
		}
	}

	return parseErrors
}

// parseInfo takes an Info object, parses it into a K8s resource and puts it into one of the 3 struct slices:
// the workload resource slice, the Service resource slice and the ConfigMaps resource slice
// It also updates the set of services to be exposed when parsing Ingress or OpenShift Routes
func (rf *resourceAccumulator) parseInfo(info *resource.Info) error {
	if info == nil || info.Object == nil {
		return fmt.Errorf("a bad Info object - Object field is Nil")
	}

	kind := info.Object.GetObjectKind().GroupVersionKind().Kind
	if !acceptedK8sTypes.MatchString(kind) {
		msg := fmt.Sprintf("skipping object with type: %s", kind)
		resourcePath := info.Source
		if resourcePath != "" {
			msg = fmt.Sprintf("in file: %s, %s", resourcePath, msg)
		}
		rf.logger.Infof(msg)
		return nil
	}

	switch kind {
	case service:
		res, err := k8sServiceFromInfo(info)
		if err != nil {
			return err
		}
		res.Resource.FilePath = info.Source
		rf.services = append(rf.services, res)
	case route:
		err := ocRouteFromInfo(info, rf.servicesToExpose)
		if err != nil {
			return err
		}
	case ingress:
		err := k8sIngressFromInfo(info, rf.servicesToExpose)
		if err != nil {
			return err
		}
	case configmap:
		res, err := k8sConfigmapFromInfo(info)
		if err != nil {
			return err
		}
		rf.configmaps = append(rf.configmaps, res)
	default:
		res, err := k8sWorkloadObjectFromInfo(info)
		if err != nil {
			return err
		}
		res.Resource.FilePath = info.Source
		rf.workloads = append(rf.workloads, res)
	}

	return nil
}

// inlineConfigMapRefsAsEnvs appends to the Envs of each given resource the ConfigMap values it is referring to
// It should only be called after ALL calls to getRelevantK8sResources successfully returned
func (rf *resourceAccumulator) inlineConfigMapRefsAsEnvs() []FileProcessingError {
	cfgMapsByName := map[string]*cfgMap{}
	for _, cm := range rf.configmaps {
		cfgMapsByName[cm.FullName] = cm
	}

	parseErrors := []FileProcessingError{}
	for _, res := range rf.workloads {
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
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name), rf.logger)
			}
		}

		// inline PodSpec->container->env->valueFrom->configMapKeyRef
		for _, cfgMapKeyRef := range res.Resource.ConfigMapKeyRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapKeyRef.Name
			if cfgMap, ok := cfgMapsByName[configmapFullName]; ok {
				if val, ok := cfgMap.Data[cfgMapKeyRef.Key]; ok {
					if netAddr, ok := networkAddressFromStr(val); ok {
						res.Resource.NetworkAddrs = append(res.Resource.NetworkAddrs, netAddr)
					}
				} else {
					err := configMapKeyNotFound(cfgMapKeyRef.Name, cfgMapKeyRef.Key, res.Resource.Name)
					parseErrors = appendAndLogNewError(parseErrors, err, rf.logger)
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name), rf.logger)
			}
		}
	}
	return parseErrors
}

// exposeServices changes the exposure of services pointed by resources such as Route or Ingress.
// This will ensure that the network policy for their workloads will allow ingress from all the cluster or from the outside internet.
// It should only be called after ALL calls to getRelevantK8sResources successfully returned
func (rf *resourceAccumulator) exposeServices() {
	for _, svc := range rf.services {
		exposedServicesInNamespace, ok := rf.servicesToExpose[svc.Resource.Namespace]
		if !ok {
			continue
		}
		if exposeExternally, ok := exposedServicesInNamespace[svc.Resource.Name]; ok {
			if exposeExternally {
				svc.Resource.ExposeExternally = true
			} else {
				svc.Resource.ExposeToCluster = true
			}
		}
	}
}
