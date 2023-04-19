package controller

import (
	"github.com/np-guard/cluster-topology-analyzer/pkg/analyzer"
	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

// inlineConfigMapRefsAsEnvs appends to the Envs of each given resource the ConfigMap values it is referring to
func inlineConfigMapRefsAsEnvs(resources []common.Resource, cfgMaps []common.CfgMap, logger Logger) []FileProcessingError {
	cfgMapsByName := map[string]common.CfgMap{}
	for cm := range cfgMaps {
		cfgMapsByName[cfgMaps[cm].FullName] = cfgMaps[cm]
	}

	parseErrors := []FileProcessingError{}
	for idx := range resources {
		res := &resources[idx]

		// inline the envFrom field in PodSpec->containers
		for _, cfgMapRef := range res.Resource.ConfigMapRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapRef
			if cfgMap, ok := cfgMapsByName[configmapFullName]; ok {
				for _, v := range cfgMap.Data {
					if analyzer.IsNetworkAddressValue(v) {
						res.Resource.NetworkAddrs = append(res.Resource.NetworkAddrs, v)
					}
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name), logger)
			}
		}

		// inline PodSpec->container->env->valueFrom->configMapKeyRef
		for _, cfgMapKeyRef := range res.Resource.ConfigMapKeyRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapKeyRef.Name
			if cfgMap, ok := cfgMapsByName[configmapFullName]; ok {
				if val, ok := cfgMap.Data[cfgMapKeyRef.Key]; ok {
					if analyzer.IsNetworkAddressValue(val) {
						res.Resource.NetworkAddrs = append(res.Resource.NetworkAddrs, val)
					}
				} else {
					err := configMapKeyNotFound(cfgMapKeyRef.Name, cfgMapKeyRef.Key, res.Resource.Name)
					parseErrors = appendAndLogNewError(parseErrors, err, logger)
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name), logger)
			}
		}
	}
	return parseErrors
}
