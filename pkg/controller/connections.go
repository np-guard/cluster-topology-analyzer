package controller

import (
	"fmt"
	"strings"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
	"go.uber.org/zap"
)

var debug = false

// This function is at the core of the topology analysis
// For each resource, it finds other resources that may use it and compiles a list of connections holding these dependencies
func discoverConnections(resources []common.Resource, links []common.Service) ([]common.Connections, error) {
	connections := []common.Connections{}
	for _, destRes := range resources {
		deploymentServices := findServices(destRes.Resource.Selectors, links)
		for _, s := range deploymentServices {
			srcRes := findSource(resources, s)
			if len(srcRes) > 0 {
				for _, r := range srcRes {
					zap.S().Debugf("source: %s target: %s link: %s", s.Resource.Name, r.Resource.Name, s.Resource.Name)
					connections = append(connections, common.Connections{Source: r, Target: destRes, Link: s})
				}
			} else {
				connections = append(connections, common.Connections{Target: destRes, Link: s}) // indicates a source-less service
			}
		}
	}
	return connections, nil
}

//areSelectorsContained returns true if selectors2 is contained in selectors1
func areSelectorsContained(selectors1 []string, selectors2 []string) bool {
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
	for _, l := range links {
		//all service selector values should be contained in the input selectors of the deployment
		res := areSelectorsContained(selectors, l.Resource.Selectors)
		if res {
			matchedSvc = append(matchedSvc, l)
		}
	}

	if debug {
		zap.S().Debugf("matched service: %v", matchedSvc)
	}
	return matchedSvc
}

// findSource returns a list of resources that are likely trying to connect to the given service
func findSource(resources []common.Resource, service common.Service) []common.Resource {
	tRes := []common.Resource{}
	for _, r := range resources {
		if debug {
			zap.S().Debugf("resource: %s", r.Resource.Name)
		}
		for _, envVal := range r.Resource.Envs {
			if strings.HasPrefix(envVal, "http://") {
				envVal = strings.TrimLeft(envVal, "http://")
			}
			if service.Resource.Name == envVal { // A match without port name
				tRes = append(tRes, r)
			}
			for _, p := range service.Resource.Network {
				serviceWithPort := fmt.Sprintf("%s:%d", service.Resource.Name, p.Port)
				if serviceWithPort == envVal {
					foundSrc := r
					//specify the used ports for target by the found src
					foundSrc.Resource.UsedPorts = []int{p.Port}
					tRes = append(tRes, foundSrc)
				}
			}
		}
	}
	return tRes
}
