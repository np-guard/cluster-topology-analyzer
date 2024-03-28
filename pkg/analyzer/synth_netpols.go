/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"reflect"
	"sort"

	core "k8s.io/api/core/v1"
	network "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	networkPolicyKind = "NetworkPolicy"
)

type deploymentConnectivity struct {
	Resource
	ingressConns []network.NetworkPolicyIngressRule
	egressConns  []network.NetworkPolicyEgressRule
}

func (deployConn *deploymentConnectivity) addIngressRule(
	peers []network.NetworkPolicyPeer, ports []network.NetworkPolicyPort) {
	rule := network.NetworkPolicyIngressRule{From: peers, Ports: ports}
	for _, existingRule := range deployConn.ingressConns {
		if reflect.DeepEqual(existingRule, rule) {
			return
		}
	}
	deployConn.ingressConns = append(deployConn.ingressConns, rule)
}

func (deployConn *deploymentConnectivity) addEgressRule(
	peers []network.NetworkPolicyPeer, ports []network.NetworkPolicyPort) {
	rule := network.NetworkPolicyEgressRule{To: peers, Ports: ports}
	for _, existingRule := range deployConn.egressConns {
		if reflect.DeepEqual(existingRule, rule) {
			return
		}
	}
	deployConn.egressConns = append(deployConn.egressConns, rule)
}

// Generate a default-deny NetworkPolicy for the given namespace
func getNsDefaultDenyPolicy(namespace string) *network.NetworkPolicy {
	policyName := "default-deny-in-namespace"
	if namespace != "" {
		policyName += "-" + namespace
	}
	return &network.NetworkPolicy{
		TypeMeta: metaV1.TypeMeta{
			Kind:       networkPolicyKind,
			APIVersion: network.SchemeGroupVersion.String(),
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      policyName,
			Namespace: namespace,
		},
		Spec: network.NetworkPolicySpec{
			PodSelector: metaV1.LabelSelector{},               // select all pods in the namespace
			Ingress:     []network.NetworkPolicyIngressRule{}, // deny all ingress
			Egress:      []network.NetworkPolicyEgressRule{},  // deny all egress
			PolicyTypes: []network.PolicyType{network.PolicyTypeIngress, network.PolicyTypeEgress},
		},
	}
}

// Generate default-deny NetworkPolicy for each namespace of the given resources
func getNsDefaultDenyPolicies(resources []*Resource) []*network.NetworkPolicy {
	denyNetpols := []*network.NetworkPolicy{}
	namespaces := map[string]bool{}
	for _, res := range resources {
		namespace := res.Resource.Namespace
		if _, ok := namespaces[namespace]; !ok {
			namespaces[namespace] = true
			denyNetpols = append(denyNetpols, getNsDefaultDenyPolicy(namespace))
		}
	}
	return denyNetpols
}

func (ps *PoliciesSynthesizer) synthNetpols(resources []*Resource, connections []*Connections) []*network.NetworkPolicy {
	deployConnectivity := determineConnectivityPerDeployment(connections)
	netpols := ps.buildNetpolPerDeployment(deployConnectivity)
	netpols = append(netpols, getNsDefaultDenyPolicies(resources)...)
	return netpols
}

func determineConnectivityPerDeployment(connections []*Connections) []*deploymentConnectivity {
	deploysConnectivity := map[string]*deploymentConnectivity{}
	for _, conn := range connections {
		srcDeploy := findOrAddDeploymentConn(conn.Source, deploysConnectivity)
		dstDeploy := findOrAddDeploymentConn(conn.Target, deploysConnectivity)
		targetPorts := toNetpolPorts(conn.Link.Resource.Network)
		if conn.Source != nil && len(conn.Source.Resource.UsedPorts) > 0 {
			targetPorts = toNetpolPorts(conn.Source.Resource.UsedPorts)
		}

		if srcDeploy != nil {
			netpolPeers := getNetpolPeers(srcDeploy, dstDeploy, conn.Link)
			if netpolPeers != nil {
				srcDeploy.addEgressRule(netpolPeers, targetPorts)
			}
		}

		if dstDeploy != nil {
			netpolPeers := getNetpolPeers(dstDeploy, srcDeploy, conn.Link)
			if netpolPeers != nil {
				dstDeploy.addIngressRule(netpolPeers, targetPorts)
			}
		}
	}

	retSlice := []*deploymentConnectivity{}
	for _, deployConn := range deploysConnectivity {
		retSlice = append(retSlice, deployConn)
	}
	// sort by name
	sort.Slice(retSlice, func(i, j int) bool {
		return retSlice[i].Resource.Resource.Name < retSlice[j].Resource.Resource.Name
	})
	return retSlice
}

func findOrAddDeploymentConn(resource *Resource, deployConns map[string]*deploymentConnectivity) *deploymentConnectivity {
	if resource == nil || resource.Resource.Name == "" {
		return nil
	}
	if deployConn, found := deployConns[resource.Resource.Name]; found {
		return deployConn
	}

	deploy := deploymentConnectivity{Resource: *resource}
	deployConns[resource.Resource.Name] = &deploy
	return &deploy
}

func getNetpolPeers(netpolDeploy, otherDeploy *deploymentConnectivity, link *Service) []network.NetworkPolicyPeer {
	switch {
	case link.Resource.ExposeExternally: // allowing traffic to/from all endpoints
		return []network.NetworkPolicyPeer{}
	case link.Resource.ExposeToCluster: // allowing traffic to/from all cluster endpoints
		return []network.NetworkPolicyPeer{{NamespaceSelector: &metaV1.LabelSelector{}}}
	case len(link.Resource.exposeToNS) > 0: // allowing traffic to/from endpoints in selected namespaces
		res := []network.NetworkPolicyPeer{}
		for _, ns := range link.Resource.exposeToNS {
			labelMap := map[string]string{core.LabelMetadataName: ns}
			peer := network.NetworkPolicyPeer{NamespaceSelector: &metaV1.LabelSelector{MatchLabels: labelMap}}
			res = append(res, peer)
		}
		return res
	case otherDeploy != nil: // allowing traffic to the specified deplotment only
		netpolPeer := network.NetworkPolicyPeer{PodSelector: getDeployConnSelector(otherDeploy)}
		if netpolDeploy.Resource.Resource.Namespace != otherDeploy.Resource.Resource.Namespace {
			if otherDeploy.Resource.Resource.Namespace != "" {
				netpolPeer.NamespaceSelector = &metaV1.LabelSelector{
					MatchLabels: map[string]string{core.LabelMetadataName: otherDeploy.Resource.Resource.Namespace},
				}
			} // if otherDeploy has no namespace specified, we assume it is in the same namespace as the netpolDeploy
		}
		return []network.NetworkPolicyPeer{netpolPeer}
	}
	return nil
}

func getDeployConnSelector(deployConn *deploymentConnectivity) *metaV1.LabelSelector {
	return &metaV1.LabelSelector{MatchLabels: deployConn.Resource.Resource.Labels}
}

func toNetpolPorts(ports []SvcNetworkAttr) []network.NetworkPolicyPort {
	netpolPorts := make([]network.NetworkPolicyPort, 0, len(ports))
	for _, port := range ports {
		protocol := port.Protocol
		if protocol == "" {
			protocol = core.ProtocolTCP
		}
		portNum := port.TargetPort
		if portNum.Type == intstr.Int && portNum.IntVal == 0 {
			portNum = intstr.FromInt(port.Port)
		}
		netpolPort := network.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &portNum,
		}
		netpolPorts = append(netpolPorts, netpolPort)
	}
	return netpolPorts
}

func (ps *PoliciesSynthesizer) buildNetpolPerDeployment(deployConnectivity []*deploymentConnectivity) []*network.NetworkPolicy {
	netpols := make([]*network.NetworkPolicy, 0, len(deployConnectivity))
	for _, deployConn := range deployConnectivity {
		if len(deployConn.egressConns) > 0 { // add a rule to allow egress DNS traffic (inside the cluster)
			allClusterPeers := []network.NetworkPolicyPeer{{NamespaceSelector: &metaV1.LabelSelector{}}}
			deployConn.addEgressRule(allClusterPeers, []network.NetworkPolicyPort{getDNSPort(ps.dnsPort)})
		}
		netpol := network.NetworkPolicy{
			TypeMeta: metaV1.TypeMeta{
				Kind:       networkPolicyKind,
				APIVersion: network.SchemeGroupVersion.String(),
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name:      deployConn.Resource.Resource.Name + "-netpol",
				Namespace: deployConn.Resource.Resource.Namespace,
			},
			Spec: network.NetworkPolicySpec{
				PodSelector: *getDeployConnSelector(deployConn),
				Ingress:     deployConn.ingressConns,
				Egress:      deployConn.egressConns,
				PolicyTypes: []network.PolicyType{network.PolicyTypeIngress, network.PolicyTypeEgress},
			},
		}
		netpols = append(netpols, &netpol)
	}
	return netpols
}

func getDNSPort(portNum int) network.NetworkPolicyPort {
	udp := core.ProtocolUDP
	port := intstr.FromInt(portNum)
	return network.NetworkPolicyPort{
		Protocol: &udp,
		Port:     &port,
	}
}

// NetpolListFromNetpolSlice converts a slice of Kubernetes NetworkPolicies to a Kubernetes NetworkPolicyList
// containing all the policies in the slice.
func NetpolListFromNetpolSlice(netpols []*network.NetworkPolicy) network.NetworkPolicyList {
	netpols2 := []network.NetworkPolicy{}
	for _, netpol := range netpols {
		netpols2 = append(netpols2, *netpol)
	}
	netpolList := network.NetworkPolicyList{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "NetworkPolicyList",
			APIVersion: network.SchemeGroupVersion.String(),
		},
		Items: netpols2,
	}

	return netpolList
}
