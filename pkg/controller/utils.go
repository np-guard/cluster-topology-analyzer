package controller

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

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
	acceptedK8sTypesRegex = fmt.Sprintf("(%s|%s|%s|%s|%s|%s|%s|%s|%s|%s)",
		pod, replicaSet, replicationController, deployment, daemonset, statefulset, job, cronJob, service, configmap)
	acceptedK8sTypes = regexp.MustCompile(acceptedK8sTypesRegex)
	yamlSuffix       = regexp.MustCompile(".ya?ml$")
)

type parsedK8sObjects struct {
	ManifestFilepath string
	DeployObjects    []deployObject
}

type deployObject struct {
	GroupKind     string
	RuntimeObject []byte
}

// return a list of yaml files under a given directory (recursively)
func searchDeploymentManifests(repoDir string) ([]string, common.FileProcessingErrorList) {
	yamls := []string{}
	errors := common.FileProcessingErrorList{}
	err := filepath.WalkDir(repoDir, func(path string, f os.DirEntry, err error) error {
		if err != nil {
			errors = append(errors, &common.FileProcessingError{Msg: fmt.Sprintf("error accessing dir: %v", err), FilePath: path})
			return filepath.SkipDir
		}
		if f != nil && !f.IsDir() && yamlSuffix.MatchString(f.Name()) {
			yamls = append(yamls, path)
		}
		return nil
	})
	if err != nil {
		walkError := common.FileProcessingError{Msg: fmt.Sprintf("error walking dir: %v", err), FilePath: repoDir}
		errors = append(errors, &walkError)
		zap.S().Error(walkError.Msg)
	}
	return yamls, errors
}

func getK8sDeploymentResources(repoDir string) ([]parsedK8sObjects, common.FileProcessingErrorList) {
	manifestFiles, fileScanErrors := searchDeploymentManifests(repoDir)
	if len(manifestFiles) == 0 {
		noYamlsError := common.FileProcessingError{Msg: "no yaml files found"}
		fileScanErrors = append(fileScanErrors, &noYamlsError)
		zap.S().Info(noYamlsError.Msg)
		return nil, fileScanErrors
	}
	parsedObjs := []parsedK8sObjects{}
	for _, mfp := range manifestFiles {
		deployObjects, err := parseK8sYaml(mfp)
		fileScanErrors = append(fileScanErrors, err...)
		if len(deployObjects) > 0 {
			manifestFilepath := mfp
			if pathSplit := strings.Split(mfp, repoDir); len(pathSplit) > 1 {
				manifestFilepath = pathSplit[1]
			}
			parsedObjs = append(parsedObjs, parsedK8sObjects{DeployObjects: deployObjects, ManifestFilepath: manifestFilepath})
		}
	}
	return parsedObjs, fileScanErrors
}

func splitByYamlDocuments(mfp string) ([]string, common.FileProcessingErrorList) {
	fileBuf, err := os.ReadFile(mfp)
	if err != nil {
		readError := common.FileProcessingError{Msg: fmt.Sprintf("error reading file: %v", err), FilePath: mfp}
		return []string{}, common.FileProcessingErrorList{&readError}
	}

	decoder := yaml.NewDecoder(bytes.NewBuffer(fileBuf))
	documents := []string{}
	fileProcessingErrors := common.FileProcessingErrorList{}
	documentID := 0
	for {
		var doc map[interface{}]interface{}
		if err := decoder.Decode(&doc); err != nil {
			if err != io.EOF {
				msg := fmt.Sprintf("YAML parsing error: %v", err)
				parseError := common.FileProcessingError{Msg: msg, FilePath: mfp, DocID: documentID}
				fileProcessingErrors = append(fileProcessingErrors, &parseError)
				zap.S().Warn(msg)
			}
			break
		}
		if len(doc) > 0 {
			out, err := yaml.Marshal(doc)
			if err != nil {
				msg := fmt.Sprintf("failed marshaling YAML document: %v", err)
				marshalError := common.FileProcessingError{Msg: msg, FilePath: mfp, DocID: documentID}
				fileProcessingErrors = append(fileProcessingErrors, &marshalError)
				zap.S().Warn(msg)
			} else {
				documents = append(documents, string(out))
			}
		}
		documentID += 1
	}
	return documents, fileProcessingErrors
}

func parseK8sYaml(mfp string) ([]deployObject, common.FileProcessingErrorList) {
	dObjs := []deployObject{}
	sepYamlFiles, fileProcessingErrors := splitByYamlDocuments(mfp)
	for docID, doc := range sepYamlFiles {
		if doc == "\n" || doc == "" {
			continue // ignore empty yaml documents
		}
		decode := scheme.Codecs.UniversalDeserializer().Decode
		_, groupVersionKind, err := decode([]byte(doc), nil, nil)
		if err != nil {
			msg := fmt.Sprintf("Yaml document is not a K8s resource: %v", err)
			fileProcessingErrors = append(fileProcessingErrors, &common.FileProcessingError{Msg: msg, FilePath: mfp, DocID: docID})
			zap.S().Warn(msg)
			continue
		}
		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			zap.S().Infof("Skipping object with type: %s", groupVersionKind.Kind)
		} else {
			d := deployObject{}
			d.GroupKind = groupVersionKind.Kind
			d.RuntimeObject = []byte(doc)
			dObjs = append(dObjs, d)
		}
	}
	return dObjs, fileProcessingErrors
}
