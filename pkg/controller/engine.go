package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	networking "k8s.io/api/networking/v1"

	"github.com/np-guard/cluster-topology-analyzer/pkg/analyzer"
	"github.com/np-guard/cluster-topology-analyzer/pkg/common"

	"go.uber.org/zap"
)

// Start : This is the entry point for the topology analysis engine.
// Based on the arguments it is given, the engine scans all YAML files,
// detects all required connection between resources and outputs a json connectivity report
// (or NetworkPolicies to allow only this connectivity)
func Start(args common.InArgs) error {
	// 1. Discover all connections between resources
	connections, err := extractConnections(args)
	if err != nil {
		return err
	}

	// 2. Write the output to a file or to stdout
	const indent = "    "
	var buf []byte
	if args.SynthNetpols != nil && *args.SynthNetpols {
		buf, err = json.MarshalIndent(synthNetpolList(connections), "", indent)
	} else {
		buf, err = json.MarshalIndent(connections, "", indent)
	}
	if err != nil {
		return err
	}
	if *args.OutputFile != "" {
		fp, err := os.Create(*args.OutputFile)
		if err != nil {
			msg := fmt.Sprintf("error creating file: %s: %v", *args.OutputFile, err)
			zap.S().Errorf(msg)
			return errors.New(msg)
		}
		_, err = fp.Write(buf)
		if err != nil {
			msg := fmt.Sprintf("error writing to file: %s: %v", *args.OutputFile, err)
			zap.S().Errorf(msg)
			return errors.New(msg)
		}
		fp.Close()
	} else {
		fmt.Printf("connection topology reports: \n ---\n%s\n---", string(buf))
	}
	return nil
}

func PoliciesFromFolderPath(fullTargetPath string) ([]*networking.NetworkPolicy, error) {
	emptyStr := ""
	args := common.InArgs{}
	args.DirPath = &fullTargetPath
	args.CommitID = &emptyStr
	args.GitBranch = &emptyStr
	args.GitURL = &emptyStr

	connections, err := extractConnections(args)
	if err != nil {
		return []*networking.NetworkPolicy{}, err
	}
	return synthNetpols(connections), nil
}

func extractConnections(args common.InArgs) ([]common.Connections, error) {
	// 1. Get all relevant resources from the repo and parse them
	dObjs := getK8sDeploymentResources(args.DirPath)
	if len(dObjs) == 0 {
		msg := "no deployment objects discovered in the repository"
		zap.S().Errorf(msg)
		return []common.Connections{}, errors.New(msg)
	}
	resources, links := parseResources(dObjs, args)

	// 2. Discover all connections between resources
	return discoverConnections(resources, links)
}

func parseResources(objs []parsedK8sObjects, args common.InArgs) ([]common.Resource, []common.Service) {
	resources := []common.Resource{}
	links := []common.Service{}
	configmaps := map[string]common.CfgMap{} // map from a configmap's full-name to its data
	for _, o := range objs {
		r, l, c := parseResource(o)
		if len(r) != 0 {
			resources = append(resources, r...)
		}
		if len(l) != 0 {
			links = append(links, l...)
		}
		for _, cfgObj := range c {
			configmaps[cfgObj.FullName] = cfgObj
		}
	}
	for idx := range resources {
		res := &resources[idx]
		res.CommitID = *args.CommitID
		res.GitBranch = *args.GitBranch
		res.GitURL = *args.GitURL

		// handle config maps data to be associated into relevant deployments resource objects
		for _, cfgMapRef := range res.Resource.ConfigMapRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapRef
			if cfgMap, ok := configmaps[configmapFullName]; ok {
				for _, v := range cfgMap.Data {
					res.Resource.Envs = append(res.Resource.Envs, v)
				}
			}
		}
		for _, cfgMapKeyRef := range res.Resource.ConfigMapKeyRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapKeyRef.Name
			if cfgMap, ok := configmaps[configmapFullName]; ok {
				if val, ok := cfgMap.Data[cfgMapKeyRef.Key]; ok {
					if analyzer.IsNetworkAddressValue(val) {
						res.Resource.Envs = append(res.Resource.Envs, val)
					}
				}
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

func parseResource(obj parsedK8sObjects) ([]common.Resource, []common.Service, []common.CfgMap) {
	links := []common.Service{}
	deployments := []common.Resource{}
	configMaps := []common.CfgMap{}

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

	return deployments, links, configMaps
}
