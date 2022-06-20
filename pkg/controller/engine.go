package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/cluster-topology-analyzer/pkg/analyzer"
	"github.com/cluster-topology-analyzer/pkg/common"
	"go.uber.org/zap"
)

//Start :
func Start(args common.InArgs) error {
	//1. Get all relevant resources from the repo and parse them
	dObjs := getK8sDeploymentResources(args.DirPath)
	if len(dObjs) == 0 {
		zap.S().Info("no deployment objects discovered in the repository")
		return nil
	}
	resources, links := parseResources(dObjs, args)

	// 2. Discover all connections between resources
	connections, _ := discoverConnections(resources, links)

	// 3. Write the output to a file or to stdout
	var buf []byte
	if args.SynthNetpols != nil && *args.SynthNetpols {
		buf = synthNetpols(connections)
	} else {
		buf, _ = json.MarshalIndent(connections, "", "    ")
	}
	if *args.OutputFile != "" {
		fp, err := os.Create(*args.OutputFile)
		if err != nil {
			msg := fmt.Sprintf("error creating file: %s: %v", *args.OutputFile, err)
			zap.S().Errorf(msg)
			return errors.New(msg)
		}
		fp.Write(buf)
		fp.Close()
	} else {
		fmt.Printf("connection topology reports: \n ---\n%s\n---", string(buf))
	}
	return nil
}

func parseResources(objs []parsedK8sObjects, args common.InArgs) ([]common.Resource, []common.Service) {
	resources := []common.Resource{}
	links := []common.Service{}
	configmaps := make(map[string][]string, 0) // map for each configmap full-name to its list of data values
	for _, o := range objs {
		r, l, c := parseResource(o)
		if len(r) != 0 {
			resources = append(resources, r...)
		}
		if len(l) != 0 {
			links = append(links, l...)
		}
		for _, cfgObj := range c {
			for k, v := range cfgObj {
				configmaps[k] = v
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
				//TODO: keep only data values with addresses of known services names
				//pattern for relevant data value: http://[svc name]:[port] or [svc-name]:[port] or [svc-name] or  http://[svc name] (implied port 80)
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
	return resources, links
}

func parseResource(obj parsedK8sObjects) ([]common.Resource, []common.Service, []common.CfgMapData) {
	links := []common.Service{}
	deployments := []common.Resource{}
	configMaps := []common.CfgMapData{}

	for _, p := range obj.DeployObjects {
		switch p.GroupKind {
		case "Service":
			res, err := analyzer.ScanK8sServiceObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				zap.S().Errorf("error scanning service object: %v", err)
				continue
			}
			res.Resource.FilePath = obj.ManifestFilepath
			links = append(links, res)
		case "ConfigMap":
			res, err := analyzer.ScanK8sConfigmapObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				zap.S().Errorf("error scanning Configmap object: %v", err)
				continue
			}
			configMaps = append(configMaps, res)
		default:
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
