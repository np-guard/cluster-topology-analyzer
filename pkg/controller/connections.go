package controller

import (
	"fmt"
	"strings"

	"github.com/cluster-topology-analyzer/pkg/common"
	"go.uber.org/zap"
)

var debug = false

func discoverConnections(resources []common.Resource, links []common.Service) ([]common.Connections, error) {
	connections := []common.Connections{}
	for _, destRes := range resources {
		c := common.Connections{}
		svc, svcFound := findService(destRes.Resource.Selectors, links)
		// if debug {
		for _, s := range svc {
			zap.S().Debugf("[]source: %s service: %s", destRes.Resource.Name, s.Resource.Name)
		}
		// }``
		if svcFound {
			for _, s := range svc {
				srcRes, srcFound := findSource(resources, s)
				if srcFound {
					for _, r := range srcRes {
						zap.S().Debugf("source: %s target: %s link: %s", s.Resource.Name, r.Resource.Name, s.Resource.Name)
						c.Source = r
						c.Target = destRes
						c.Link = s
						connections = append(connections, c)
					}
				} else {
					c.Target = destRes
					c.Link = s
					connections = append(connections, c)
				}
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

//findService returns a list of services from input links matching the input selectors, and a bool
//flag indicating if matching services were found
func findService(selectors []string, links []common.Service) ([]common.Service, bool) {
	var matchedSvc []common.Service
	var found bool
	//TODO: refer to namespaces - the matching services and input deployment should be in the same namespace
	for _, l := range links {
		//all service selector values should be contained in the input selectors of the deployment
		res := areSelectorsContained(selectors, l.Resource.Selectors)
		if res {
			matchedSvc = append(matchedSvc, l)
			found = true
		}
	}

	if debug {
		zap.S().Debugf("matched service: %v", matchedSvc)
	}
	return matchedSvc, found
}

func findSource(resources []common.Resource, service common.Service) ([]common.Resource, bool) {
	tRes := []common.Resource{}
	found := false
	for _, p := range service.Resource.Network {
		ep := fmt.Sprintf("%s:%d", service.Resource.Name, p.Port)
		for _, r := range resources {
			if debug {
				zap.S().Debugf("resource: %s", r.Resource.Name)
			}
			for _, e := range r.Resource.Envs {
				if strings.HasPrefix(e, "http://") {
					e = strings.TrimLeft(e, "http://")
				}
				if debug {
					zap.S().Debugf("deployment env: %s", e)
				}
				if strings.Compare(ep, e) == 0 {
					foundSrc := r
					//specify the used ports for target by the found src
					foundSrc.Resource.UsedPorts = []int{p.Port}
					tRes = append(tRes, foundSrc)
					found = true
				}
			}
		}
	}
	return tRes, found
}
