package controller

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"os"

	networking "k8s.io/api/networking/v1"

	"github.com/np-guard/cluster-topology-analyzer/pkg/analyzer"
	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
	"go.uber.org/zap"
)

type ErrMode string

const (
	Warn         ErrMode = "warning"
	Strict       ErrMode = "strict"
	SilentIgnore ErrMode = "ignore"
)

type InArgs struct {
	DirPath              *string
	ManifestFiles        []string
	GitURL               *string
	GitBranch            *string
	CommitID             *string
	OutputFile           *string
	SynthNetpols         *bool
	UseStrict            bool
	SilentlyIgnoreErrors bool
}

// Start : This is the entry point for the topology analysis engine.
// Based on the arguments it is given, the engine scans all YAML files,
// detects all required connection between resources and outputs a json connectivity report
// (or NetworkPolicies to allow only this connectivity)
func Start(args InArgs, errMode ErrMode) error {
	// 1. Discover all connections between resources
	discoveredManifests, err := findManifests(*args.DirPath)
	if err != nil {
		return errors.Wrap(err, "extracting manifests failed")
	}
	connections, err := connectionsFromFiles(discoveredManifests, errMode)
	if err != nil {
		return errors.Wrap(err, "extracting connections failed")
	}
	return writeOut(connections, *args.OutputFile, *args.SynthNetpols)
}

func writeOut(connections []common.Connections, outFile string, synth bool) error {
	// 2. Write the output to a file or to stdout
	const indent = "    "
	var err error
	var buf []byte
	if synth {
		buf, err = json.MarshalIndent(synthNetpolList(connections), "", indent)
	} else {
		buf, err = json.MarshalIndent(connections, "", indent)
	}
	if err != nil {
		return err
	}
	if outFile != "" {
		fp, err := os.Create(outFile)
		if err != nil {
			msg := fmt.Sprintf("error creating file: %s: %v", outFile, err)
			zap.S().Errorf(msg)
			return errors.New(msg)
		}
		_, err = fp.Write(buf)
		if err != nil {
			msg := fmt.Sprintf("error writing to file: %s: %v", outFile, err)
			zap.S().Errorf(msg)
			return errors.New(msg)
		}
		if err := fp.Close(); err != nil {
			return err
		}
	} else {
		fmt.Printf("connection topology reports: \n ---\n%s\n---", string(buf))
	}
	return nil
}

// PoliciesFromFolderPath calculates network policies from manifest files discovered while walking the folder tree
func PoliciesFromFolderPath(fullTargetPath string, mode ErrMode) ([]*networking.NetworkPolicy, error) {
	discoveredManifests, err := findManifests(fullTargetPath)
	if err != nil {
		return []*networking.NetworkPolicy{}, err
	}
	return PoliciesFromFiles(discoveredManifests, mode)
}

// PoliciesFromFiles calculates network policies from manifest files provided explicitly.
// Returns error on any issues with reading the files
func PoliciesFromFiles(files []string, mode ErrMode) ([]*networking.NetworkPolicy, error) {
	connections, err := connectionsFromFiles(files, mode)
	if err != nil {
		return []*networking.NetworkPolicy{}, errors.Wrap(err, "failed extracting connections from k8s objects")
	}
	return synthNetpols(connections), nil
}

func connectionsFromFiles(files []string, mode ErrMode) ([]common.Connections, error) {
	if len(files) == 0 {
		return []common.Connections{}, fmt.Errorf("no input files provided")
	}
	k8sDeployments := getK8sDeploymentResources(files, mode)
	for _, m := range k8sDeployments {
		if m.fileReadingError != nil {
			err := errors.Wrapf(m.fileReadingError, "unable to read file '%s'", m.ManifestFilepath)
			if mode == Warn {
				zap.S().Warn(err)
			}
			if mode == Strict {
				zap.S().Error(err)
				return []common.Connections{}, err
			}
		}
		for _, obj := range m.DeployObjects {
			if obj.yamlDocDecodeError != nil {
				err := errors.Wrapf(m.fileReadingError, "unable to parse k8s object from file '%s'", m.ManifestFilepath)
				if mode == Warn {
					zap.S().Warn(err)
				}
				if mode == Strict {
					zap.S().Error(err)
					return []common.Connections{}, err
				}
			}
		}
	}
	// INFO: Deployments extracted successfully
	return extractConnections(k8sDeployments, "", "", "")
}

func findManifests(dirPath string) ([]string, error) {
	if dirPath == "" {
		return []string{}, fmt.Errorf("missing folder path for manifests discovery")
	}
	discoveredManifests, err := searchDeploymentManifests(dirPath)
	if err != nil {
		return []string{}, fmt.Errorf("error walking directory tree while searching for manifests")
	} else if len(discoveredManifests) == 0 {
		return []string{}, fmt.Errorf("found 0 manifests")
	}
	return discoveredManifests, nil
}

func extractConnections(dObjs []yamlK8sObjects, commitID, gitBranch, gitURL string) ([]common.Connections, error) {
	if len(dObjs) == 0 {
		msg := "no deployment found in the repository"
		zap.S().Errorf(msg)
		return []common.Connections{}, errors.New(msg)
	}
	resources, links := parseResources(dObjs, commitID, gitBranch, gitURL)

	// 2. Discover all connections between resources
	return discoverConnections(resources, links)
}

func parseResources(objs []yamlK8sObjects, commitID, gitBranch, gitURL string) ([]common.Resource, []common.Service) {
	resources := make([]common.Resource, 0)
	links := make([]common.Service, 0)
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
		res.CommitID = commitID
		res.GitBranch = gitBranch
		res.GitURL = gitURL

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
		links[idx].CommitID = commitID
		links[idx].GitBranch = gitBranch
		links[idx].GitURL = gitURL
	}
	return resources, links
}

func parseResource(obj yamlK8sObjects) ([]common.Resource, []common.Service, []common.CfgMap) {
	links := make([]common.Service, 0)
	deployments := make([]common.Resource, 0)
	configMaps := make([]common.CfgMap, 0)

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
