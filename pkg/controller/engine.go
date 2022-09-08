package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/np-guard/cluster-topology-analyzer/pkg/analyzer"
	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

// Start : This is the entry point for the topology analysis engine.
// Based on the arguments it is given, the engine scans all YAML files,
// detects all required connection between resources and outputs a json connectivity report
// (or NetworkPolicies to allow only this connectivity)
func Start(args common.InArgs) error {
	// 1. Discover all connections between resources
	connections, fileScanErrors := extractConnections(args, false)
	if len(fileScanErrors) > 0 {
		return fmt.Errorf("errors in processing input files: %v", fileScanErrors)
	}

	// 2. Write the output to a file or to stdout
	const indent = "    "
	var buf []byte
	var err error
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
			msg := fmt.Sprintf("error creating file: %s", *args.OutputFile)
			activeLogger.Errorf(err, msg)
			return errors.New(msg)
		}
		_, err = fp.Write(buf)
		if err != nil {
			msg := fmt.Sprintf("error writing to file: %s", *args.OutputFile)
			activeLogger.Errorf(err, msg)
			return errors.New(msg)
		}
		fp.Close()
	} else {
		fmt.Printf("connection topology reports: \n ---\n%s\n---", string(buf))
	}
	return nil
}

func extractConnections(args common.InArgs, stopOn1stErr bool) ([]common.Connections, []FileProcessingError) {
	// 1. Get all relevant resources from the repo and parse them
	dObjs, fileErrors := getK8sDeploymentResources(*args.DirPath, stopOn1stErr)
	if stopProcessing(stopOn1stErr, fileErrors) {
		return nil, fileErrors
	}
	if len(dObjs) == 0 {
		fileErrors = appendAndLogNewError(fileErrors, noK8sResourcesFound())
		return []common.Connections{}, fileErrors
	}

	resources, links, parseErrors := parseResources(dObjs, args)
	fileErrors = append(fileErrors, parseErrors...)
	if stopProcessing(stopOn1stErr, fileErrors) {
		return nil, fileErrors
	}

	// 2. Discover all connections between resources
	return discoverConnections(resources, links), fileErrors
}

func parseResources(objs []parsedK8sObjects, args common.InArgs) ([]common.Resource, []common.Service, []FileProcessingError) {
	resources := []common.Resource{}
	links := []common.Service{}
	configmaps := map[string]common.CfgMap{} // map from a configmap's full-name to its data
	parseErrors := []FileProcessingError{}
	for _, o := range objs {
		r, l, c, e := parseResource(o)
		resources = append(resources, r...)
		links = append(links, l...)
		parseErrors = append(parseErrors, e...)
		for _, cfgObj := range c {
			configmaps[cfgObj.FullName] = cfgObj
		}
	}
	for idx := range resources {
		res := &resources[idx]

		// handle config maps data to be associated into relevant deployments resource objects
		for _, cfgMapRef := range res.Resource.ConfigMapRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapRef
			if cfgMap, ok := configmaps[configmapFullName]; ok {
				for _, v := range cfgMap.Data {
					if analyzer.IsNetworkAddressValue(v) {
						res.Resource.Envs = append(res.Resource.Envs, v)
					}
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name))
			}
		}
		for _, cfgMapKeyRef := range res.Resource.ConfigMapKeyRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapKeyRef.Name
			if cfgMap, ok := configmaps[configmapFullName]; ok {
				if val, ok := cfgMap.Data[cfgMapKeyRef.Key]; ok {
					if analyzer.IsNetworkAddressValue(val) {
						res.Resource.Envs = append(res.Resource.Envs, val)
					}
				} else {
					parseErrors = appendAndLogNewError(parseErrors, configMapKeyNotFound(cfgMapKeyRef.Name, cfgMapKeyRef.Key, res.Resource.Name))
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name))
			}
		}
	}

	return resources, links, parseErrors
}

func parseResource(obj parsedK8sObjects) ([]common.Resource, []common.Service, []common.CfgMap, []FileProcessingError) {
	links := []common.Service{}
	deployments := []common.Resource{}
	configMaps := []common.CfgMap{}
	parseErrors := []FileProcessingError{}

	for _, p := range obj.DeployObjects {
		switch p.GroupKind {
		case service:
			res, err := analyzer.ScanK8sServiceObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, obj.ManifestFilepath, err))
				continue
			}
			res.Resource.FilePath = obj.ManifestFilepath
			links = append(links, res)
		case configmap:
			res, err := analyzer.ScanK8sConfigmapObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, obj.ManifestFilepath, err))
				continue
			}
			configMaps = append(configMaps, res)
		default:
			res, err := analyzer.ScanK8sWorkloadObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, obj.ManifestFilepath, err))
				continue
			}
			res.Resource.FilePath = obj.ManifestFilepath
			deployments = append(deployments, res)
		}
	}

	return deployments, links, configMaps, parseErrors
}
