package controller

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"io"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"path/filepath"
	"regexp"
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

// yamlK8sObjects represents k8s objects from a single yaml file
// (which may contain 0..N yaml documents)
type yamlK8sObjects struct {
	ManifestFilepath string
	DeployObjects    []deployObject
	fileReadingError error
	yamlParseError   error
}

type deployObject struct {
	GroupKind          string
	RuntimeObject      []byte
	yamlDocDecodeError error
}

func searchDeploymentManifests(repoDir string) ([]string, error) {
	yamls := make([]string, 0)
	err := filepath.WalkDir(repoDir, func(path string, f os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if f != nil && !f.IsDir() {
			r, err := regexp.MatchString(`^.*\.y(a)?ml$`, f.Name())
			if err == nil && r {
				yamls = append(yamls, path)
			}
		}
		return nil
	})
	if err != nil {
		return yamls, err
	}
	return yamls, nil
}

func getK8sDeploymentResources(manifestFiles []string, mode ErrMode) []yamlK8sObjects {
	parsedObjs := make([]yamlK8sObjects, 0)
	for _, mfp := range manifestFiles {
		p := yamlK8sObjects{
			ManifestFilepath: mfp,
			DeployObjects:    make([]deployObject, 0),
			fileReadingError: nil,
			yamlParseError:   nil,
		}
		filebuf, err := os.ReadFile(mfp)
		if err != nil {
			p.fileReadingError = errors.Wrap(err, "error reading file")
		} else {
			p.DeployObjects, p.yamlParseError = parseK8sYaml(filebuf, mode)
		}
		parsedObjs = append(parsedObjs, p)
	}
	return parsedObjs
}

func splitByYamlDocuments(data []byte, mode ErrMode) ([]string, error) {
	decoder := yaml.NewDecoder(bytes.NewBuffer(data))
	documents := make([]string, 0)
	for {
		var doc map[interface{}]interface{}
		if err := decoder.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			if mode == Warn {
				zap.S().Warn(err)
			}
			if mode == Strict {
				zap.S().Error(err)
				return documents, errors.Wrapf(err, "document decode failed")
			}
		}
		if len(doc) > 0 {
			out, err := yaml.Marshal(doc)
			if err != nil {
				if mode == Warn {
					zap.S().Warn(err)
				}
				if mode == Strict {
					zap.S().Error(err)
					return documents, errors.Wrapf(err, "marshalling yamls failed")
				}
			}
			strOut := string(out)
			documents = append(documents, strOut)
		}
	}
	return documents, nil
}

func parseK8sYaml(fileR []byte, mode ErrMode) ([]deployObject, error) {
	acceptedK8sTypes := regexp.MustCompile(fmt.Sprintf("(%s|%s|%s|%s|%s|%s|%s|%s|%s|%s)",
		pod, replicaSet, replicationController, deployment, daemonset, statefulset, job, cronJob, service, configmap))
	sepYamlFiles, err := splitByYamlDocuments(fileR, mode)
	if err != nil {
		return []deployObject{}, err
	}
	dObjs := make([]deployObject, 0, len(sepYamlFiles))
	for _, f := range sepYamlFiles {
		if f == "\n" || f == "" {
			// ignore empty yaml documents
			continue
		}
		decode := scheme.Codecs.UniversalDeserializer().Decode
		_, groupVersionKind, err := decode([]byte(f), nil, nil)
		d := deployObject{}
		if err != nil {
			d.yamlDocDecodeError = err
		}
		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			zap.S().Infof("Skipping object with type: %s", groupVersionKind.Kind)
		} else {
			d.GroupKind = groupVersionKind.Kind
			d.RuntimeObject = []byte(f)
			dObjs = append(dObjs, d)
		}
	}
	return dObjs, nil
}
