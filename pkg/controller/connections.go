package controller

import (
	"fmt"
	"strings"

	"github.ibm.com/gitsecure-net-top/pkg/common"
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

func findService(selectors []string, links []common.Service) ([]common.Service, bool) {
	var matchedSvc []common.Service
	var found bool
	for _, s := range selectors {
		for _, l := range links {
			for _, ls := range l.Resource.Selectors {
				if strings.Compare(s, ls) == 0 {
					matchedSvc = append(matchedSvc, l)
					found = true
					break
				}
				// if found {
				// 	break
				// }
			}
			// if found {
			// 	break
			// }
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
				if debug {
					zap.S().Debugf("deployment env: %s", e)
				}
				if strings.Compare(ep, e) == 0 {
					tRes = append(tRes, r)
					found = true
				}
			}
		}
	}
	return tRes, found
}
