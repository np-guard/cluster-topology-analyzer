/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"k8s.io/cli-runtime/pkg/resource"

	"github.com/np-guard/netpol-analyzer/pkg/netpol/manifests/fsscanner"

	"github.com/np-guard/cluster-topology-analyzer/pkg/analyzer"
	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
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
	yamlSuffix       = regexp.MustCompile(".ya?ml$")
)

// resourceFinder is used to locate all relevant K8s resources in given file-system directories
// and to convert them into the internal structs, used for later processing.
type resourceFinder struct {
	logger       Logger
	stopOn1stErr bool
	walkFn       WalkFunction // for customizing directory scan

	workloads        []*common.Resource      // accumulates all workload resources found
	services         []*common.Service       // accumulates all service resources found
	configmaps       []*common.CfgMap        // accumulates all ConfigMap resources found
	servicesToExpose common.ServicesToExpose // stores which services should be later exposed
}

func newResourceFinder(logger Logger, failFast bool, walkFn WalkFunction) *resourceFinder {
	res := resourceFinder{logger: logger, stopOn1stErr: failFast, walkFn: walkFn}

	res.servicesToExpose = common.ServicesToExpose{}

	return &res
}

// getRelevantK8sResources is the main function of resourceFinder.
// It scans a given directory using walkFn, looking for all yaml files. It then breaks each yaml into its documents
// and extracts all K8s resources that are relevant for connectivity analysis.
// The resources are stored in the struct, separated to workloads, services and configmaps
func (rf *resourceFinder) getRelevantK8sResources(repoDir string) []FileProcessingError {
	manifestFiles, fileScanErrors := rf.searchForManifests(repoDir)
	if stopProcessing(rf.stopOn1stErr, fileScanErrors) {
		return fileScanErrors
	}
	if len(manifestFiles) == 0 {
		fileScanErrors = appendAndLogNewError(fileScanErrors, noYamlsFound(), rf.logger)
		return fileScanErrors
	}

	for _, mfp := range manifestFiles {
		relMfp := pathWithoutBaseDir(mfp, repoDir)
		errs := rf.parseK8sYaml(mfp, relMfp)
		fileScanErrors = append(fileScanErrors, errs...)
		if stopProcessing(rf.stopOn1stErr, fileScanErrors) {
			return fileScanErrors
		}
	}

	return fileScanErrors
}

// searchForManifests returns a list of YAML files under a given directory (recursively)
func (rf *resourceFinder) searchForManifests(repoDir string) ([]string, []FileProcessingError) {
	yamls := []string{}
	errors := []FileProcessingError{}
	err := rf.walkFn(repoDir, func(path string, f os.DirEntry, err error) error {
		if err != nil {
			errors = appendAndLogNewError(errors, failedAccessingDir(path, err, path != repoDir), rf.logger)
			if stopProcessing(rf.stopOn1stErr, errors) {
				return err
			}
			return filepath.SkipDir
		}
		if f != nil && !f.IsDir() && yamlSuffix.MatchString(f.Name()) {
			yamls = append(yamls, path)
		}
		return nil
	})
	if err != nil {
		rf.logger.Errorf(err, "Error walking directory")
	}
	return yamls, errors
}

// parseK8sYaml takes a YAML file and attempts to parse each of its documents into
// one of the relevant k8s resources
func (rf *resourceFinder) parseK8sYaml(mfp, relMfp string) []FileProcessingError {
	infos, errs := fsscanner.GetResourceInfosFromDirPath([]string{mfp}, true, rf.stopOn1stErr)
	fileProcessingErrors := []FileProcessingError{}
	for _, err := range errs {
		fileProcessingErrors = appendAndLogNewError(fileProcessingErrors, failedReadingFile(mfp, err), rf.logger)
		if stopProcessing(rf.stopOn1stErr, fileProcessingErrors) {
			return fileProcessingErrors
		}
	}

	for _, info := range infos {
		err := rf.parseInfo(info)
		if err != nil {
			kind := info.Object.GetObjectKind().GroupVersionKind().Kind
			fileProcessingErrors = appendAndLogNewError(fileProcessingErrors, failedScanningResource(kind, relMfp, err), rf.logger)
		}
	}

	return fileProcessingErrors
}

// parseInfo takes an Info object, parses it into a K8s resource and puts it into one of the 3 struct slices:
// the workload resource slice, the Service resource slice and the ConfigMaps resource slice
// It also updates the set of services to be exposed when parsing Ingress or OpenShift Routes
func (rf *resourceFinder) parseInfo(info *resource.Info) error {
	kind := info.Object.GetObjectKind().GroupVersionKind().Kind
	if !acceptedK8sTypes.MatchString(kind) {
		resourcePath := info.Source
		rf.logger.Infof("in file: %s, skipping object with type: %s", resourcePath, kind)
		return nil
	}

	switch kind {
	case service:
		res, err := analyzer.ScanK8sServiceInfo(info)
		if err != nil {
			return err
		}
		res.Resource.FilePath = info.Source
		rf.services = append(rf.services, res)
	case route:
		err := analyzer.ScanOCRouteObjectFromInfo(info, rf.servicesToExpose)
		if err != nil {
			return err
		}
	case ingress:
		err := analyzer.ScanIngressObjectFromInfo(info, rf.servicesToExpose)
		if err != nil {
			return err
		}
	case configmap:
		res, err := analyzer.ScanK8sConfigmapInfo(info)
		if err != nil {
			return err
		}
		rf.configmaps = append(rf.configmaps, res)
	default:
		res, err := analyzer.ScanK8sWorkloadObjectFromInfo(info)
		if err != nil {
			return err
		}
		res.Resource.FilePath = info.Source
		rf.workloads = append(rf.workloads, res)
	}

	return nil
}

// returns a file path without its prefix base dir
func pathWithoutBaseDir(path, baseDir string) string {
	if path == baseDir { // baseDir is actually a file...
		return filepath.Base(path) // return just the file name
	}

	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return path
	}
	return relPath
}

// inlineConfigMapRefsAsEnvs appends to the Envs of each given resource the ConfigMap values it is referring to
// It should only be called after ALL calls to getRelevantK8sResources successfully returned
func (rf *resourceFinder) inlineConfigMapRefsAsEnvs() []FileProcessingError {
	cfgMapsByName := map[string]*common.CfgMap{}
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
					if netAddr, ok := analyzer.NetworkAddressValue(v); ok {
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
					if netAddr, ok := analyzer.NetworkAddressValue(val); ok {
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
func (rf *resourceFinder) exposeServices() {
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
