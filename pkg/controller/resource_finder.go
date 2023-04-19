package controller

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes/scheme"

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
)

var (
	acceptedK8sTypesRegex = fmt.Sprintf("(^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$)",
		pod, replicaSet, replicationController, deployment, daemonSet, statefulSet, job, cronJob, service, configmap)
	acceptedK8sTypes = regexp.MustCompile(acceptedK8sTypesRegex)
	yamlSuffix       = regexp.MustCompile(".ya?ml$")
	decoder          = scheme.Codecs.UniversalDeserializer()
)

// rawK8sResource stores a raw K8s resource and its kind for later parsing
type rawK8sResource struct {
	GroupKind     string
	RuntimeObject []byte
}

// resourceFinder is used to locate all relevant K8s resources in a given file-system directory
type resourceFinder struct {
	logger       Logger
	stopOn1stErr bool
	walkFn       WalkFunction // for customizing directory scan

	workloads  []common.Resource
	services   []common.Service
	configmaps []common.CfgMap
}

// getRelevantK8sResources is the main function of resourceFinder.
// It scans a given directory using walkFn, looking for all yaml files. It then breaks each yaml into its documents
// and extracts all K8s resources that are relevant for connectivity analysis.
// The resources are returned separated to workloads, services and configmaps
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
		rawK8sResources, err := rf.parseK8sYaml(mfp)
		fileScanErrors = append(fileScanErrors, err...)
		if stopProcessing(rf.stopOn1stErr, fileScanErrors) {
			return fileScanErrors
		}
		manifestFilePath := pathWithoutBaseDir(mfp, repoDir)
		errs := rf.parseResources(rawK8sResources, manifestFilePath)
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
func (rf *resourceFinder) splitByYamlDocuments(mfp string) ([]string, []FileProcessingError) {
	fileBuf, err := os.ReadFile(mfp)
	if err != nil {
		return []string{}, appendAndLogNewError(nil, failedReadingFile(mfp, err), rf.logger)
	}

	decoder := yaml.NewDecoder(bytes.NewBuffer(fileBuf))
	documents := []string{}
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
			documents = append(documents, string(out))
		}
		documentID += 1
	}
	return documents, nil
}

// parseK8sYaml takes a YAML document and checks if it stands for a relevant K8s resource.
// If yes, it puts it into a rawK8sResource and appends it to the result.
func (rf *resourceFinder) parseK8sYaml(mfp string) ([]rawK8sResource, []FileProcessingError) {
	dObjs := []rawK8sResource{}
	yamlDocs, fileProcessingErrors := rf.splitByYamlDocuments(mfp)
	if stopProcessing(rf.stopOn1stErr, fileProcessingErrors) {
		return nil, fileProcessingErrors
	}

	for docID, doc := range yamlDocs {
		_, groupVersionKind, err := decoder.Decode([]byte(doc), nil, nil)
		if err != nil {
			fileProcessingErrors = appendAndLogNewError(fileProcessingErrors, notK8sResource(mfp, docID, err), rf.logger)
			continue
		}
		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			rf.logger.Infof("in file: %s, document: %d, skipping object with type: %s", mfp, docID, groupVersionKind.Kind)
		} else {
			dObjs = append(dObjs, rawK8sResource{GroupKind: groupVersionKind.Kind, RuntimeObject: []byte(doc)})
		}
	}
	return dObjs, fileProcessingErrors
}

// parseResources takes raw K8s resources in a file and breaks them into 3 separate slices:
// a slice with workload resources, a slice with Service resources, and a slice with ConfigMaps resources
func (rf *resourceFinder) parseResources(rawK8sResources []rawK8sResource, manifestFilePath string) []FileProcessingError {
	parseErrors := []FileProcessingError{}
	for _, p := range rawK8sResources {
		switch p.GroupKind {
		case service:
			res, err := analyzer.ScanK8sServiceObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, manifestFilePath, err), rf.logger)
				continue
			}
			res.Resource.FilePath = manifestFilePath
			rf.services = append(rf.services, res)
		case configmap:
			res, err := analyzer.ScanK8sConfigmapObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, manifestFilePath, err), rf.logger)
				continue
			}
			rf.configmaps = append(rf.configmaps, res)
		default:
			res, err := analyzer.ScanK8sWorkloadObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, manifestFilePath, err), rf.logger)
				continue
			}
			res.Resource.FilePath = manifestFilePath
			rf.workloads = append(rf.workloads, res)
		}
	}

	return parseErrors
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
