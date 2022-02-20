package controller

import (
	"encoding/json"
	"fmt"
	"os"

	"github.ibm.com/gitsecure-net-top/pkg/analyzer"
	"github.ibm.com/gitsecure-net-top/pkg/common"
	"go.uber.org/zap"
)

//Start :
func Start(args common.InArgs) error {

	//1. Get the deployment objects from the repo
	dObjs := getK8sDeploymentResources(args.DirPath)
	if len(dObjs) == 0 {
		zap.S().Info("no deployment objects discovered in the repository")
		return nil
	}
	resources := []common.Resource{}
	links := []common.Service{}
	configmaps := make(map[string][]string, 0) // map for each configmap full-name to its list of data values
	for _, o := range dObjs {
		r, l, c := parseResouce(o)
		if len(r) != 0 {
			resources = append(resources, r...)
		}
		if len(l) != 0 {
			links = append(links, l...)
		}
		if len(c) != 0 {
			for _, cfgObj := range c {
				for k, v := range cfgObj {
					configmaps[k] = v
				}
			}
		}
		// zap.S().Debugf("resources: %v \n\n links: %v", resources, links)
	}
	for idx := range resources {
		resources[idx].CommitID = *args.CommitID
		resources[idx].GitBranch = *args.GitBranch
		resources[idx].GitURL = *args.GitURL

		//handle config maps data to be associated into relevant deployments resource objects
		if resources[idx].Resource.ConfigMapRef != "" {
			configmapFullName := resources[idx].Resource.Namespace + "/" + resources[idx].Resource.ConfigMapRef
			if data, ok := configmaps[configmapFullName]; ok {
				//add to d Envs the values from data
				// TODO: keep only data values with addresses of known services names
				// pattern for relevant data value: http://[svc name]:[port] or [svc-name]:[port] or [svc-name] or  http://[svc name] (implied port 80)
				//The port is optional when it is the default port for a given protocol (e.g., HTTP=80).
				resources[idx].Resource.Envs = append(resources[idx].Resource.Envs, data...)
			}
		}
	}
	for idx := range links {
		links[idx].CommitID = *args.CommitID
		links[idx].GitBranch = *args.GitBranch
		links[idx].GitURL = *args.GitURL
	}
	connections, _ := discoverConnections(resources, links)
	printToStdOut := true
	buf, _ := json.MarshalIndent(connections, "", "    ")
	if *args.OutputFile != "" {
		fp, err := os.Create(*args.OutputFile)
		if err != nil {
			zap.S().Debugf("error creating file: %s: %v", *args.OutputFile, err)
		} else {
			printToStdOut = false
			fp.Write(buf)
			fp.Close()
		}
	}
	if printToStdOut {
		fmt.Printf("connection topology reports: \n ---\n%s\n---", string(buf))
	}
	return nil
}

func parseResouce(obj parsedK8sObjects) ([]common.Resource, []common.Service, []common.CfgMapData) {
	links := []common.Service{}
	deployments := []common.Resource{}
	configMaps := []common.CfgMapData{}
	//configmaps := make(map[string][]string, 0)

	for _, p := range obj.DeployObjects {
		if p.GroupKind == "Service" {
			res, err := analyzer.ScanK8sServiceObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				zap.S().Errorf("error scanning service object: %v", err)
				continue
			}
			res.Resource.FilePath = obj.ManifestFilepath
			links = append(links, res)
		} else if p.GroupKind == "ConfigMap" {
			res, err := analyzer.ScanK8sConfigmapObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				zap.S().Errorf("error scanning Configmap object: %v", err)
				continue
			}
			//configmaps[fullName] = data
			configMaps = append(configMaps, res)
		} else {
			res, err := analyzer.ScanK8sDeployObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				zap.S().Debugf("Skipping object with type: %s", p.GroupKind)
				continue
			}
			res.Resource.FilePath = obj.ManifestFilepath
			deployments = append(deployments, res)
		}
	}

	// zap.S().Debugf("[1]resources: %d links: %d", len(deployments), len(links))
	return deployments, links, configMaps
}
