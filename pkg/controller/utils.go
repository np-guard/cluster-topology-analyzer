package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.uber.org/zap"
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

type parsedK8sObjects struct {
	ManifestFilepath string
	DeployObjects    []deployObject
}

type deployObject struct {
	GroupKind     string
	RuntimeObject []byte
}

func searchDeploymentManifests(repoDir *string) []string {
	yamls := []string{}
	err := filepath.WalkDir(*repoDir, func(path string, f os.DirEntry, err error) error {
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
		zap.S().Errorf("Error is searching for manifests: %v", err)
	}
	return yamls
}

func getK8sDeploymentResources(repoDir *string) []parsedK8sObjects {
	manifestFiles := searchDeploymentManifests(repoDir)
	if len(manifestFiles) == 0 {
		zap.S().Info("no deployment manifest found")
		return nil
	}
	parsedObjs := []parsedK8sObjects{}
	for _, mfp := range manifestFiles {
		filebuf, err := os.ReadFile(mfp)
		if err != nil {
			continue
		}
		p := parsedK8sObjects{}
		p.ManifestFilepath = mfp
		if pathSplit := strings.Split(mfp, *repoDir); len(pathSplit) > 1 {
			p.ManifestFilepath = pathSplit[1]
		}
		p.DeployObjects = parseK8sYaml(filebuf)
		parsedObjs = append(parsedObjs, p)
	}
	return parsedObjs
}

func parseK8sYaml(fileR []byte) []deployObject {
	dObjs := []deployObject{}
	acceptedK8sTypes := regexp.MustCompile(fmt.Sprintf("(%s|%s|%s|%s|%s|%s|%s|%s|%s|%s)",
		pod, replicaSet, replicationController, deployment, daemonset, statefulset, job, cronJob, service, configmap))
	fileAsString := string(fileR)
	sepYamlFiles := regexp.MustCompile(`---\s`).Split(fileAsString, -1)
	for _, f := range sepYamlFiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}
		decode := scheme.Codecs.UniversalDeserializer().Decode
		_, groupVersionKind, err := decode([]byte(f), nil, nil)
		if err != nil {
			continue
		}
		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			zap.S().Infof("Skipping object with type: %s", groupVersionKind.Kind)
		} else {
			d := deployObject{}
			d.GroupKind = groupVersionKind.Kind
			d.RuntimeObject = []byte(f)
			dObjs = append(dObjs, d)
		}
	}
	return dObjs
}

// Exists Check whether a file with a given path exists
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
