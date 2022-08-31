package controller

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes/scheme"
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
func searchDeploymentManifests(repoDir string, stopOn1stErr bool) ([]string, []FileProcessingError) {
	yamls := []string{}
	errors := []FileProcessingError{}
	err := filepath.WalkDir(repoDir, func(path string, f os.DirEntry, err error) error {
		if err != nil {
			errors = append(errors, *failedAccessingDir(path, err, path != repoDir))
			if returnOn1StError(stopOn1stErr, errors) {
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
		errors = append(errors, *failedWalkDir(repoDir, err))
	}
	return yamls, errors
}

func getK8sDeploymentResources(repoDir string, stopOn1stErr bool) ([]parsedK8sObjects, []FileProcessingError) {
	manifestFiles, fileScanErrors := searchDeploymentManifests(repoDir, stopOn1stErr)
	if returnOn1StError(stopOn1stErr, fileScanErrors) {
		return nil, fileScanErrors
	}
	if len(manifestFiles) == 0 {
		fileScanErrors = append(fileScanErrors, *noYamlsFound())
		return nil, fileScanErrors
	}

	parsedObjs := []parsedK8sObjects{}
	for _, mfp := range manifestFiles {
		deployObjects, err := parseK8sYaml(mfp, stopOn1stErr)
		fileScanErrors = append(fileScanErrors, err...)
		if returnOn1StError(stopOn1stErr, fileScanErrors) {
			return nil, fileScanErrors
		}
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

func splitByYamlDocuments(mfp string) ([]string, []FileProcessingError) {
	fileBuf, err := os.ReadFile(mfp)
	if err != nil {
		return []string{}, []FileProcessingError{*failedReadingFile(mfp, err)}
	}

	decoder := yaml.NewDecoder(bytes.NewBuffer(fileBuf))
	documents := []string{}
	documentID := 0
	for {
		var doc map[interface{}]interface{}
		if err := decoder.Decode(&doc); err != nil {
			if err != io.EOF {
				return documents, []FileProcessingError{*malformedYamlDoc(mfp, documentID, err)}
			}
			break
		}
		if len(doc) > 0 {
			out, err := yaml.Marshal(doc)
			if err != nil {
				return documents, []FileProcessingError{*malformedYamlDoc(mfp, documentID, err)}
			}
			documents = append(documents, string(out))
		}
		documentID += 1
	}
	return documents, nil
}

func parseK8sYaml(mfp string, stopOn1stErr bool) ([]deployObject, []FileProcessingError) {
	dObjs := []deployObject{}
	sepYamlFiles, fileProcessingErrors := splitByYamlDocuments(mfp)
	if returnOn1StError(stopOn1stErr, fileProcessingErrors) {
		return nil, fileProcessingErrors
	}

	for docID, doc := range sepYamlFiles {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		_, groupVersionKind, err := decode([]byte(doc), nil, nil)
		if err != nil {
			fileProcessingErrors = append(fileProcessingErrors, *notK8sResource(mfp, docID, err))
			continue
		}
		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			activeLogger.Infof("Skipping object with type: %s", groupVersionKind.Kind)
		} else {
			d := deployObject{}
			d.GroupKind = groupVersionKind.Kind
			d.RuntimeObject = []byte(doc)
			dObjs = append(dObjs, d)
		}
	}
	return dObjs, fileProcessingErrors
}

func returnOn1StError(stopOn1stErr bool, errs []FileProcessingError) bool {
	return stopOn1stErr && len(errs) > 0 && (errs[len(errs)-1].IsSevere() || errs[len(errs)-1].IsFatal())
}
