package controller

import (
	"reflect"
	"sort"
	"strings"

	"github.com/cluster-topology-analyzer/pkg/common"
	core "k8s.io/api/core/v1"
	network "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type DeploymentConnectivity struct {
	common.Resource
	ingress_conns []network.NetworkPolicyIngressRule
	egress_conns  []network.NetworkPolicyEgressRule
}

func (deployConn *DeploymentConnectivity) addIngressRule(
	peers []network.NetworkPolicyPeer, ports []network.NetworkPolicyPort) {
	rule := network.NetworkPolicyIngressRule{From: peers, Ports: ports}
	for _, existingRule := range deployConn.ingress_conns {
		if reflect.DeepEqual(existingRule, rule) {
			return
		}
	}
	deployConn.ingress_conns = append(deployConn.ingress_conns, rule)
}

func (deployConn *DeploymentConnectivity) addEgressRule(
	peers []network.NetworkPolicyPeer, ports []network.NetworkPolicyPort) {
	rule := network.NetworkPolicyEgressRule{To: peers, Ports: ports}
	for _, existingRule := range deployConn.egress_conns {
		if reflect.DeepEqual(existingRule, rule) {
			return
		}
	}
	deployConn.egress_conns = append(deployConn.egress_conns, rule)
}

func synthNetpols(connections []common.Connections) []network.NetworkPolicy {
	deployConnectivity := determineConnectivityPerDeployment(connections)
	netpols := buildNetpolPerDeployment(deployConnectivity)
	return netpols
}

func determineConnectivityPerDeployment(connections []common.Connections) []*DeploymentConnectivity {
	deploysConnectivity := map[string]*DeploymentConnectivity{}
	for _, conn := range connections {
		srcDeploy := findOrAddDeploymentConn(conn.Source, deploysConnectivity)
		dstDeploy := findOrAddDeploymentConn(conn.Target, deploysConnectivity)
		target_ports := toNetpolPorts(conn.Link.Resource.Network) // TODO: filter by src ports

		egressNetpolPeer := []network.NetworkPolicyPeer{{PodSelector: getDeployConnSelector(dstDeploy)}}
		srcDeploy.addEgressRule(egressNetpolPeer, target_ports)
		var ingressNetpolPeer []network.NetworkPolicyPeer
		if len(conn.Source.Resource.Name) == 0 {
			ingressNetpolPeer = append(ingressNetpolPeer, network.NetworkPolicyPeer{})
		} else if conn.Link.Resource.Type != "LoadBalancer" {
			netpolPeer := network.NetworkPolicyPeer{PodSelector: getDeployConnSelector(srcDeploy)}
			ingressNetpolPeer = append(ingressNetpolPeer, netpolPeer)
		}
		dstDeploy.addIngressRule(ingressNetpolPeer, target_ports)
	}

	retSlice := []*DeploymentConnectivity{}
	for _, deployConn := range deploysConnectivity {
		retSlice = append(retSlice, deployConn)
	}
	// sort by name
	sort.Slice(retSlice, func(i, j int) bool {
		return retSlice[i].Resource.Resource.Name < retSlice[j].Resource.Resource.Name
	})
	return retSlice
}

func findOrAddDeploymentConn(resource common.Resource, deployConns map[string]*DeploymentConnectivity) *DeploymentConnectivity {
	if deployConn, found := deployConns[resource.Resource.Name]; found {
		return deployConn
	}

	deploy := DeploymentConnectivity{Resource: resource}
	deployConns[resource.Resource.Name] = &deploy
	return &deploy
}

func getDeployConnSelector(deployConn *DeploymentConnectivity) *metaV1.LabelSelector {
	selectorsMap := map[string]string{}
	for _, selector := range deployConn.Resource.Resource.Selectors {
		key := selector[:strings.Index(selector, ":")]
		value := selector[strings.Index(selector, ":")+1:]
		selectorsMap[key] = value
	}
	return &metaV1.LabelSelector{MatchLabels: selectorsMap}
}

func toNetpolPorts(ports []common.SvcNetworkAttr) []network.NetworkPolicyPort {
	var netpolPorts []network.NetworkPolicyPort
	for _, port := range ports {
		protocol := toCoreProtocol(port.Protocol)
		portNum := intstr.FromInt(port.TargetPort)
		netpolPort := network.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &portNum,
		}
		netpolPorts = append(netpolPorts, netpolPort)
	}
	return netpolPorts
}

func toCoreProtocol(protocol string) core.Protocol {
	switch protocol {
	case "TCP":
		return core.ProtocolTCP
	case "UDP":
		return core.ProtocolUDP
	case "SCTP":
		return core.ProtocolSCTP
	default:
		return core.ProtocolTCP
	}
}

func buildNetpolPerDeployment(deployConnectivity []*DeploymentConnectivity) []network.NetworkPolicy {
	var netpols []network.NetworkPolicy
	for _, deployConn := range deployConnectivity {
		if len(deployConn.egress_conns) > 0 {
			deployConn.addEgressRule(nil, []network.NetworkPolicyPort{getDnsPort()})
		}
		netpol := network.NetworkPolicy{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "NetworkPolicy",
				APIVersion: "networking.k8s.io/v1",
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name:      deployConn.Resource.Resource.Name + "-netpol",
				Namespace: deployConn.Resource.Resource.Namespace,
			},
			Spec: network.NetworkPolicySpec{
				PodSelector: *getDeployConnSelector(deployConn),
				Ingress:     deployConn.ingress_conns,
				Egress:      deployConn.egress_conns,
				PolicyTypes: []network.PolicyType{network.PolicyTypeIngress, network.PolicyTypeEgress},
			},
		}
		netpols = append(netpols, netpol)
	}
	return netpols
}

func getDnsPort() network.NetworkPolicyPort {
	udp := core.ProtocolUDP
	port53 := intstr.FromInt(53)
	return network.NetworkPolicyPort{
		Protocol: &udp,
		Port:     &port53,
	}
}
