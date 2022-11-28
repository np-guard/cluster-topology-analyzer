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
)

// K8s resources that are relevant for connectivity analysis
const (
	pod                   string = "Pod"
	replicaSet            string = "ReplicaSet"
	replicationController string = "ReplicationController"
	deployment            string = "Deployment"
	statefulset           string = "StatefulSet"
	daemonset             string = "DaemonSet"
	job                   string = "Job"
	cronJob               string = "CronTab"
	service               string = "Service"
	configmap             string = "ConfigMap"
)

var (
	acceptedK8sTypesRegex = fmt.Sprintf("(^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$|^%s$)",
		pod, replicaSet, replicationController, deployment, daemonset, statefulset, job, cronJob, service, configmap)
	acceptedK8sTypes = regexp.MustCompile(acceptedK8sTypesRegex)
	yamlSuffix       = regexp.MustCompile(".ya?ml$")
)

// rawResourcesInFile represents a single YAML file with multiple K8s resources
type rawResourcesInFile struct {
	ManifestFilepath string
	rawK8sResources  []rawK8sResource
}

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
}

// getRelevantK8sResources is the main function of resourceFinder.
// It scans a given directory using walkFn, looking for all yaml files. It then breaks each yaml into its documents
// and extracts all K8s resources that are relevant for connectivity analysis.
// The result is stored in a slice of rawResourcesInFile (one per yaml file), each containing a slice of rawK8sResource
func (rf *resourceFinder) getRelevantK8sResources(repoDir string) ([]rawResourcesInFile, []FileProcessingError) {
	manifestFiles, fileScanErrors := rf.searchForManifests(repoDir)
	if stopProcessing(rf.stopOn1stErr, fileScanErrors) {
		return nil, fileScanErrors
	}
	if len(manifestFiles) == 0 {
		fileScanErrors = appendAndLogNewError(fileScanErrors, noYamlsFound(), rf.logger)
		return nil, fileScanErrors
	}

	parsedObjs := []rawResourcesInFile{}
	for _, mfp := range manifestFiles {
		rawK8sResources, err := rf.parseK8sYaml(mfp)
		fileScanErrors = append(fileScanErrors, err...)
		if stopProcessing(rf.stopOn1stErr, fileScanErrors) {
			return nil, fileScanErrors
		}
		if len(rawK8sResources) > 0 {
			manifestFilePath := pathWithoutBaseDir(mfp, repoDir)
			parsedObjs = append(parsedObjs, rawResourcesInFile{rawK8sResources: rawK8sResources, ManifestFilepath: manifestFilePath})
		}
	}
	return parsedObjs, fileScanErrors
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
	sepYamlFiles, fileProcessingErrors := rf.splitByYamlDocuments(mfp)
	if stopProcessing(rf.stopOn1stErr, fileProcessingErrors) {
		return nil, fileProcessingErrors
	}

	for docID, doc := range sepYamlFiles {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		_, groupVersionKind, err := decode([]byte(doc), nil, nil)
		if err != nil {
			fileProcessingErrors = appendAndLogNewError(fileProcessingErrors, notK8sResource(mfp, docID, err), rf.logger)
			continue
		}
		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			rf.logger.Infof("in file: %s, document: %d, skipping object with type: %s", mfp, docID, groupVersionKind.Kind)
		} else {
			d := rawK8sResource{}
			d.GroupKind = groupVersionKind.Kind
			d.RuntimeObject = []byte(doc)
			dObjs = append(dObjs, d)
		}
	}
	return dObjs, fileProcessingErrors
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
