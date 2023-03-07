package controller

import (
	"github.com/np-guard/cluster-topology-analyzer/pkg/analyzer"
	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

// resourceParser encapsulates the code for extracting workload resources and service resources from raw YAML documents
type resourceParser struct {
	logger Logger
}

// parseResources takes a slice of raw resources per file, and returns a slice with all workload resources
// and a slice with all Service resources. It also flattens all references a workload makes to ConfigMaps.
func (rp *resourceParser) parseResources(objs []rawResourcesInFile) ([]common.Resource, []common.Service, []FileProcessingError) {
	resources := []common.Resource{}
	links := []common.Service{}
	configmaps := map[string]common.CfgMap{} // map from a configmap's full-name to its data
	parseErrors := []FileProcessingError{}
	for _, o := range objs {
		r, l, c, e := rp.parseResource(o)
		resources = append(resources, r...)
		links = append(links, l...)
		parseErrors = append(parseErrors, e...)
		for _, cfgObj := range c {
			configmaps[cfgObj.FullName] = cfgObj
		}
	}

	errs := rp.inlineConfigMapRefsAsEnvs(resources, configmaps)
	parseErrors = append(parseErrors, errs...)

	return resources, links, parseErrors
}

// parseResource takes raw K8s resources in a file and breaks them into 3 separate slices:
// a slice with workload resources, a slice with Service resources, and a slice with ConfigMaps resources
func (rp *resourceParser) parseResource(obj rawResourcesInFile) (
	[]common.Resource,
	[]common.Service,
	[]common.CfgMap,
	[]FileProcessingError,
) {
	links := []common.Service{}
	deployments := []common.Resource{}
	configMaps := []common.CfgMap{}
	parseErrors := []FileProcessingError{}

	for _, p := range obj.rawK8sResources {
		switch p.GroupKind {
		case service:
			res, err := analyzer.ScanK8sServiceObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, obj.ManifestFilepath, err), rp.logger)
				continue
			}
			res.Resource.FilePath = obj.ManifestFilepath
			links = append(links, res)
		case configmap:
			res, err := analyzer.ScanK8sConfigmapObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, obj.ManifestFilepath, err), rp.logger)
				continue
			}
			configMaps = append(configMaps, res)
		default:
			res, err := analyzer.ScanK8sWorkloadObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, obj.ManifestFilepath, err), rp.logger)
				continue
			}
			res.Resource.FilePath = obj.ManifestFilepath
			deployments = append(deployments, res)
		}
	}

	return deployments, links, configMaps, parseErrors
}

// inlineConfigMapRefsAsEnvs appends to the Envs of each given resource the ConfigMap values it is referring to
func (rp *resourceParser) inlineConfigMapRefsAsEnvs(resources []common.Resource, cfgMaps map[string]common.CfgMap) []FileProcessingError {
	parseErrors := []FileProcessingError{}
	for idx := range resources {
		res := &resources[idx]

		// inline the envFrom field in PodSpec->containers
		for _, cfgMapRef := range res.Resource.ConfigMapRefs {
			configmapFullName := namespacedName(res.Resource.Namespace, cfgMapRef)
			if cfgMap, ok := cfgMaps[configmapFullName]; ok {
				for _, v := range cfgMap.Data {
					if analyzer.IsNetworkAddressValue(v) {
						res.Resource.NetworkAddrs = append(res.Resource.NetworkAddrs, v)
					}
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name), rp.logger)
			}
		}

		// inline PodSpec->container->env->valueFrom->configMapKeyRef
		for _, cfgMapKeyRef := range res.Resource.ConfigMapKeyRefs {
			configmapFullName := namespacedName(res.Resource.Namespace, cfgMapKeyRef.Name)
			if cfgMap, ok := cfgMaps[configmapFullName]; ok {
				if val, ok := cfgMap.Data[cfgMapKeyRef.Key]; ok {
					if analyzer.IsNetworkAddressValue(val) {
						res.Resource.NetworkAddrs = append(res.Resource.NetworkAddrs, val)
					}
				} else {
					err := configMapKeyNotFound(cfgMapKeyRef.Name, cfgMapKeyRef.Key, res.Resource.Name)
					parseErrors = appendAndLogNewError(parseErrors, err, rp.logger)
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name), rp.logger)
			}
		}
	}
	return parseErrors
}
