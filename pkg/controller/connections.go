package controller

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

var debug = false

// This function is at the core of the topology analysis
// For each resource, it finds other resources that may use it and compiles a list of connections holding these dependencies
func discoverConnections(resources []common.Resource, links []common.Service) ([]common.Connections, error) {
	connections := []common.Connections{}
	for destResIdx := range resources {
		destRes := &resources[destResIdx]
		deploymentServices := findServices(destRes.Resource.Selectors, links)
		for svcIdx := range deploymentServices {
			svc := &deploymentServices[svcIdx]
			srcRes := findSource(resources, svc)
			if len(srcRes) > 0 {
				for _, r := range srcRes {
					zap.S().Debugf("source: %s target: %s link: %s", svc.Resource.Name, r.Resource.Name, svc.Resource.Name)
					connections = append(connections, common.Connections{Source: r, Target: destRes, Link: svc})
				}
			} else {
				connections = append(connections, common.Connections{Target: destRes, Link: svc}) // indicates a source-less service
			}
		}
	}
	return connections, nil
}

// areSelectorsContained returns true if selectors2 is contained in selectors1
func areSelectorsContained(selectors1, selectors2 []string) bool {
	elementMap := make(map[string]string)
	for _, s := range selectors1 {
		elementMap[s] = ""
	}
	for _, val := range selectors2 {
		_, ok := elementMap[val]
		if !ok {
			return false
		}
	}
	return true
}

// findServices returns a list of services that may be in front of a given deployment (represented by its selectors)
func findServices(selectors []string, links []common.Service) []common.Service {
	var matchedSvc []common.Service
	//TODO: refer to namespaces - the matching services and input deployment should be in the same namespace
	for linkIdx := range links {
		link := &links[linkIdx]
		// all service selector values should be contained in the input selectors of the deployment
		res := areSelectorsContained(selectors, link.Resource.Selectors)
		if res {
			matchedSvc = append(matchedSvc, *link)
		}
	}

	if debug {
		zap.S().Debugf("matched service: %v", matchedSvc)
	}
	return matchedSvc
}

// findSource returns a list of resources that are likely trying to connect to the given service
func findSource(resources []common.Resource, service *common.Service) []*common.Resource {
	tRes := []*common.Resource{}
	for resIdx := range resources {
		res := &resources[resIdx]
		for _, envVal := range res.Resource.Envs {
			envVal = strings.TrimPrefix(envVal, "http://")
			if service.Resource.Name == envVal { // A match without port name
				tRes = append(tRes, res)
			}
			for _, p := range service.Resource.Network {
				serviceWithPort := fmt.Sprintf("%s:%d", service.Resource.Name, p.Port)
				if serviceWithPort == envVal {
					foundSrc := *res
					// specify the used ports for target by the found src
					foundSrc.Resource.UsedPorts = []int{p.Port}
					tRes = append(tRes, &foundSrc)
				}
			}
		}
	}
	return tRes
}
