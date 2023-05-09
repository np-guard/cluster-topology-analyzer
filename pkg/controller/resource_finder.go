package controller

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	ocapiv1 "github.com/openshift/api"
	ocroute "github.com/openshift/api/route/v1"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

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
)

var (
	acceptedK8sTypesRegex = fmt.Sprintf("(^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$)",
		pod, replicaSet, replicationController, deployment, daemonSet, statefulSet, job, cronJob, service, configmap, route)
	acceptedK8sTypes = regexp.MustCompile(acceptedK8sTypesRegex)
	yamlSuffix       = regexp.MustCompile(".ya?ml$")
)

// resourceFinder is used to locate all relevant K8s resources in given file-system directories
// and to convert them into the internal structs, used for later processing.
type resourceFinder struct {
	logger       Logger
	stopOn1stErr bool
	walkFn       WalkFunction // for customizing directory scan

	resourceDecoder runtime.Decoder

	workloads  []common.Resource // accumulates all workload resources found
	services   []common.Service  // accumulates all service resources found
	configmaps []common.CfgMap   // accumulates all ConfigMap resources found
	routes     []*ocroute.Route
}

func newResourceFinder(logger Logger, failFast bool, walkFn WalkFunction) *resourceFinder {
	res := resourceFinder{logger: logger, stopOn1stErr: failFast, walkFn: walkFn}

	scheme := runtime.NewScheme()
	Codecs := serializer.NewCodecFactory(scheme)
	_ = ocapiv1.InstallKube(scheme) // returned error is ignored - seems to be always nil
	_ = ocapiv1.Install(scheme)     // returned error is ignored - seems to be always nil
	res.resourceDecoder = Codecs.UniversalDeserializer()

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

// splitByYamlDocuments takes a YAML file and returns a slice containing its documents as raw text
func (rf *resourceFinder) splitByYamlDocuments(mfp string) ([][]byte, []FileProcessingError) {
	fileBuf, err := os.ReadFile(mfp)
	if err != nil {
		return nil, appendAndLogNewError(nil, failedReadingFile(mfp, err), rf.logger)
	}

	decoder := yaml.NewDecoder(bytes.NewBuffer(fileBuf))
	documents := [][]byte{}
	documentID := 0
	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err != nil {
			if err != io.EOF {
				return documents, appendAndLogNewError(nil, malformedYamlDoc(mfp, 0, documentID, err), rf.logger)
			}
			break
		}
		if len(doc.Content) > 0 && doc.Content[0].Kind == yaml.MappingNode {
			out, err := yaml.Marshal(doc.Content[0])
			if err != nil {
				return documents, appendAndLogNewError(nil, malformedYamlDoc(mfp, doc.Line, documentID, err), rf.logger)
			}
			documents = append(documents, out)
		}
		documentID += 1
	}
	return documents, nil
}

// parseK8sYaml takes a YAML file and attempts to parse each of its documents into
// one of the relevant k8s resources
func (rf *resourceFinder) parseK8sYaml(mfp, relMfp string) []FileProcessingError {
	yamlDocs, fileProcessingErrors := rf.splitByYamlDocuments(mfp)
	if stopProcessing(rf.stopOn1stErr, fileProcessingErrors) {
		return fileProcessingErrors
	}

	for docID, doc := range yamlDocs {
		_, groupVersionKind, err := rf.resourceDecoder.Decode(doc, nil, nil)
		if err != nil {
			fileProcessingErrors = appendAndLogNewError(fileProcessingErrors, notK8sResource(relMfp, docID, err), rf.logger)
			continue
		}
		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			rf.logger.Infof("in file: %s, document: %d, skipping object with type: %s", relMfp, docID, groupVersionKind.Kind)
		} else {
			kind := groupVersionKind.Kind
			err = rf.parseResource(kind, doc, relMfp)
			if err != nil {
				fileProcessingErrors = appendAndLogNewError(fileProcessingErrors, failedScanningResource(kind, relMfp, err), rf.logger)
			}
		}
	}
	return fileProcessingErrors
}

// parseResource takes a yaml document, parses it into a K8s resource and puts it into one of the 3 struct slices:
// the workload resource slice, the Service resource slice, and the ConfigMaps resource slice
func (rf *resourceFinder) parseResource(kind string, yamlDoc []byte, manifestFilePath string) error {
	switch kind {
	case service:
		res, err := analyzer.ScanK8sServiceObject(kind, yamlDoc)
		if err != nil {
			return err
		}
		res.Resource.FilePath = manifestFilePath
		rf.services = append(rf.services, res)
	case route:
		res, err := analyzer.ScanOCRouteObject(kind, yamlDoc)
		if err != nil {
			return err
		}
		rf.routes = append(rf.routes, res)
	case configmap:
		res, err := analyzer.ScanK8sConfigmapObject(kind, yamlDoc)
		if err != nil {
			return err
		}
		rf.configmaps = append(rf.configmaps, res)
	default:
		res, err := analyzer.ScanK8sWorkloadObject(kind, yamlDoc)
		if err != nil {
			return err
		}
		res.Resource.FilePath = manifestFilePath
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
	for cm := range rf.configmaps {
		cfgMapsByName[rf.configmaps[cm].FullName] = &rf.configmaps[cm]
	}

	parseErrors := []FileProcessingError{}
	for idx := range rf.workloads {
		res := &rf.workloads[idx]

		// inline the envFrom field in PodSpec->containers
		for _, cfgMapRef := range res.Resource.ConfigMapRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapRef
			if cfgMap, ok := cfgMapsByName[configmapFullName]; ok {
				for _, v := range cfgMap.Data {
					if analyzer.IsNetworkAddressValue(v) {
						res.Resource.NetworkAddrs = append(res.Resource.NetworkAddrs, v)
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
					if analyzer.IsNetworkAddressValue(val) {
						res.Resource.NetworkAddrs = append(res.Resource.NetworkAddrs, val)
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

// exposeServicesWithRoutes changes the type of services pointed by a Route resource to "LoadBalancer".
// This will ensure that the network policy for their workloads will allow ingress from the outside internet.
// It should only be called after ALL calls to getRelevantK8sResources successfully returned
func (rf *resourceFinder) exposeServicesWithRoutes() {
	// First, build a map from namespace name to a set of service names exposed by routes in this namespace
	servicesExposedByRoutes := map[string]map[string]bool{}
	for _, route := range rf.routes {
		exposedServicesInNamespace, ok := servicesExposedByRoutes[route.Namespace]
		if !ok {
			servicesExposedByRoutes[route.Namespace] = map[string]bool{}
			exposedServicesInNamespace = servicesExposedByRoutes[route.Namespace]
		}
		exposedServicesInNamespace[route.Spec.To.Name] = true
		for _, backend := range route.Spec.AlternateBackends {
			exposedServicesInNamespace[backend.Name] = true
		}
	}

	// Now, change the type of all services that appear in this map to "LoadBalancer"
	for svcIdx := range rf.services {
		svc := &rf.services[svcIdx]
		exposedServicesInNamespace, ok := servicesExposedByRoutes[svc.Resource.Namespace]
		if !ok {
			continue
		}
		if _, ok = exposedServicesInNamespace[svc.Resource.Name]; ok {
			svc.Resource.Type = corev1.ServiceTypeLoadBalancer
		}
	}
}
