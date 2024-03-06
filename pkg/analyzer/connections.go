/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type connectionExtractor struct {
	resources []*Resource
	links     []*Service
	logger    Logger
}

// This function is at the core of the topology analysis
// For each resource, it finds other resources that may use it and compiles a list of connections holding these dependencies
func (ce *connectionExtractor) discoverConnections() []*Connections {
	connections := []*Connections{}
	for _, destRes := range ce.resources {
		deploymentServices := ce.findServices(destRes)
		ce.logger.Debugf("services matched to %v: %v", destRes.Resource.Name, deploymentServices)
		for _, svc := range deploymentServices {
			srcRes := ce.findSource(svc)
			if len(srcRes) > 0 {
				for _, r := range srcRes {
					if !r.equals(destRes) {
						ce.logger.Debugf("source: %s target: %s link: %s", r.Resource.Name, destRes.Resource.Name, svc.Resource.Name)
						connections = append(connections, &Connections{Source: r, Target: destRes, Link: svc})
					}
				}
			} else {
				connections = append(connections, &Connections{Target: destRes, Link: svc}) // indicates a source-less service
			}
		}
	}
	return connections
}

// areSelectorsContained returns true if selectors2 is contained in selectors1
func areSelectorsContained(selectors1 map[string]string, selectors2 []string) bool {
	elementMap := make(map[string]string)
	for k, v := range selectors1 {
		s := fmt.Sprintf("%s:%s", k, v)
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

// findServices returns a list of services that may be in front of a given workload resource
func (ce *connectionExtractor) findServices(resource *Resource) []*Service {
	var matchedSvc []*Service
	for _, link := range ce.links {
		if link.Resource.Namespace != resource.Resource.Namespace {
			continue
		}
		// all service selector values should be contained in the input selectors of the deployment
		res := areSelectorsContained(resource.Resource.Labels, link.Resource.Selectors)
		if res {
			matchedSvc = append(matchedSvc, link)
		}
	}

	return matchedSvc
}

// findSource returns a list of resources that are likely trying to connect to the given service
func (ce *connectionExtractor) findSource(service *Service) []*Resource {
	tRes := []*Resource{}
	for _, resource := range ce.resources {
		serviceAddresses := getPossibleServiceAddresses(service, resource)
		foundSrc := *resource // We copy the resource so we can specify the ports used by the source found
		matched := false
		for _, envVal := range resource.Resource.NetworkAddrs {
			match, port := envValueMatchesService(envVal, service, serviceAddresses)
			if match {
				matched = true
				if port.Port > 0 {
					foundSrc.Resource.UsedPorts = append(foundSrc.Resource.UsedPorts, port)
				}
			}
		}
		if matched {
			tRes = append(tRes, &foundSrc)
		}
	}
	return tRes
}

func getPossibleServiceAddresses(service *Service, resource *Resource) []string {
	svcAddresses := []string{}
	if service.Resource.Namespace != "" {
		serviceDotNamespace := fmt.Sprintf("%s.%s", service.Resource.Name, service.Resource.Namespace)
		svcAddresses = append(svcAddresses, serviceDotNamespace, serviceDotNamespace+".svc.cluster.local")
	}
	if service.Resource.Namespace == resource.Resource.Namespace { // both service and resource live in the same namespace
		svcAddresses = append(svcAddresses, service.Resource.Name)
	}

	return svcAddresses
}

func envValueMatchesService(envVal string, service *Service, serviceAddresses []string) (bool, SvcNetworkAttr) {
	// first look for matches without specified port
	for _, svcAddress := range serviceAddresses {
		if svcAddress == envVal {
			return true, SvcNetworkAttr{} // this means no specified port
		}
	}

	// Now look for matches that have port specified
	for _, p := range service.Resource.Network {
		for _, svcAddress := range serviceAddresses {
			serviceWithPort := fmt.Sprintf("%s:%d", svcAddress, p.Port)
			if envVal == serviceWithPort {
				return true, p
			}
		}
	}
	return false, SvcNetworkAttr{}
}

const (
	srcDstDelim        = "->"
	endpointsPortDelim = "|"
	commentToken       = "#"
	wildcardToken      = "*"
	endpointParts      = 3
)

type workloadAndService struct {
	resource *Resource
	service  *Service
}

func (ce *connectionExtractor) connectionsFromFile(filename string) ([]*Connections, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	conns := []*Connections{}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNum += 1
		if line == "" || strings.HasPrefix(line, commentToken) {
			continue
		}
		lineConns, err := ce.parseConnectionLine(line, lineNum)
		if err != nil {
			return nil, err
		}
		conns = append(conns, lineConns...)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return conns, nil
}

func (ce *connectionExtractor) parseConnectionLine(line string, lineNum int) ([]*Connections, error) {
	// Take only the part before # starts a comment
	parts := strings.Split(line, commentToken)
	if len(parts) == 0 {
		return nil, syntaxError("unexpected comment", lineNum)
	}

	line = parts[0]

	parts = strings.Split(line, srcDstDelim)
	if len(parts) != 2 {
		return nil, syntaxError("connection line must have exactly one -> separator", lineNum)
	}

	src := strings.TrimSpace(parts[0])
	srcWorkloads, err := ce.parseEndpoints(src, lineNum)
	if err != nil {
		return nil, err
	}

	parts = strings.Split(parts[1], endpointsPortDelim)
	if len(parts) == 0 {
		return nil, syntaxError("missing destination", lineNum)
	}
	if len(parts) > 2 {
		return nil, syntaxError("connection line must have at most one | separator", lineNum)
	}
	dst := strings.TrimSpace(parts[0])
	dstWorkloads, err := ce.parseEndpoints(dst, lineNum)
	if err != nil {
		return nil, err
	}

	conns := []*Connections{}
	for _, srcWl := range srcWorkloads {
		for _, dstWl := range dstWorkloads {
			ce.logger.Debugf("src: %+v, dst: %+v\n", *srcWl.resource, *dstWl.resource)
			conns = append(conns, &Connections{
				Source: srcWl.resource,
				Target: dstWl.resource,
				Link:   dstWl.service,
			})
		}
	}
	return conns, nil
}

func (ce *connectionExtractor) parseEndpoints(endpoint string, lineNum int) ([]workloadAndService, error) {
	parts := strings.Split(endpoint, "/")
	if len(parts) != endpointParts {
		return nil, syntaxError("source and destination must be of the form namespace/kind/name", lineNum)
	}
	ns, kind, name := parts[0], parts[1], parts[2]
	kind = strings.ToUpper(kind[:1]) + kind[1:] // Capitalize kind's first letter

	var res []workloadAndService
	switch kind {
	case service:
		res = ce.getMatchingServices(ns, name)
	case wildcardToken:
		res = append(ce.getMatchingServices(ns, name), ce.getMatchingWorkloads(ns, kind, name)...)
	default:
		res = ce.getMatchingWorkloads(ns, kind, name)
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("no matching endpoints for %s in the provided manifests", endpoint)
	}
	return res, nil
}

func (ce *connectionExtractor) getMatchingServices(ns, name string) []workloadAndService {
	services := []workloadAndService{}
	for _, svc := range ce.links {
		if strMatch(svc.Resource.Namespace, ns) && strMatch(svc.Resource.Name, name) {
			services = append(services, ce.workloadsOfSvc(svc)...)
		}
	}
	return services
}

func (ce *connectionExtractor) workloadsOfSvc(svc *Service) []workloadAndService {
	svcWorkloads := []workloadAndService{}
	for _, workload := range ce.resources {
		if workload.Resource.Namespace == svc.Resource.Namespace &&
			areSelectorsContained(workload.Resource.Labels, svc.Resource.Selectors) {
			svcWorkloads = append(svcWorkloads, workloadAndService{workload, svc})
		}
	}
	return svcWorkloads
}

func (ce *connectionExtractor) getMatchingWorkloads(ns, kind, name string) []workloadAndService {
	workloads := []workloadAndService{}
	for _, workload := range ce.resources {
		if strMatch(workload.Resource.Namespace, ns) && strMatch(workload.Resource.Kind, kind) &&
			strMatch(workload.Resource.Name, name) {
			services := ce.findServices(workload)
			if len(services) == 0 {
				ce.logger.Infof("workload %s is not exposed by any service", workload.Resource.Name)
			}
			for _, svc := range services {
				workloads = append(workloads, workloadAndService{workload, svc})
			}
		}
	}
	return workloads
}

func strMatch(str, pattern string) bool {
	return pattern == wildcardToken || str == pattern
}

func syntaxError(errorStr string, lineNum int) error {
	return fmt.Errorf("syntax error in line %d: %s", lineNum, errorStr)
}
