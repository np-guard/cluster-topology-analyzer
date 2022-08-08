package controller

import (
	"reflect"
	"sort"
	"strings"

	core "k8s.io/api/core/v1"
	network "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

const dnsPort = 53

type DeploymentConnectivity struct {
	common.Resource
	ingressConns []network.NetworkPolicyIngressRule
	egressConns  []network.NetworkPolicyEgressRule
}

func (deployConn *DeploymentConnectivity) addIngressRule(
	peers []network.NetworkPolicyPeer, ports []network.NetworkPolicyPort) {
	rule := network.NetworkPolicyIngressRule{From: peers, Ports: ports}
	for _, existingRule := range deployConn.ingressConns {
		if reflect.DeepEqual(existingRule, rule) {
			return
		}
	}
	deployConn.ingressConns = append(deployConn.ingressConns, rule)
}

func (deployConn *DeploymentConnectivity) addEgressRule(
	peers []network.NetworkPolicyPeer, ports []network.NetworkPolicyPort) {
	rule := network.NetworkPolicyEgressRule{To: peers, Ports: ports}
	for _, existingRule := range deployConn.egressConns {
		if reflect.DeepEqual(existingRule, rule) {
			return
		}
	}
	deployConn.egressConns = append(deployConn.egressConns, rule)
}

func synthNetpols(connections []common.Connections) []*network.NetworkPolicy {
	deployConnectivity := determineConnectivityPerDeployment(connections)
	netpols := buildNetpolPerDeployment(deployConnectivity)
	return netpols
}

func determineConnectivityPerDeployment(connections []common.Connections) []*DeploymentConnectivity {
	deploysConnectivity := map[string]*DeploymentConnectivity{}
	for idx := range connections {
		conn := &connections[idx]
		srcDeploy := findOrAddDeploymentConn(conn.Source, deploysConnectivity)
		dstDeploy := findOrAddDeploymentConn(conn.Target, deploysConnectivity)
		targetPorts := toNetpolPorts(conn.Link.Resource.Network) // TODO: filter by src ports

		egressNetpolPeer := []network.NetworkPolicyPeer{{PodSelector: getDeployConnSelector(dstDeploy)}}
		if srcDeploy != nil {
			srcDeploy.addEgressRule(egressNetpolPeer, targetPorts)
		}

		if conn.Link.Resource.Type == "LoadBalancer" || conn.Link.Resource.Type == "NodePort" {
			dstDeploy.addIngressRule([]network.NetworkPolicyPeer{}, targetPorts) // in these cases we want to allow traffic from all sources
		} else if conn.Source != nil {
			netpolPeer := network.NetworkPolicyPeer{PodSelector: getDeployConnSelector(srcDeploy)}
			dstDeploy.addIngressRule([]network.NetworkPolicyPeer{netpolPeer}, targetPorts) // allow traffic only from this specific source
		}
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

func findOrAddDeploymentConn(resource *common.Resource, deployConns map[string]*DeploymentConnectivity) *DeploymentConnectivity {
	if resource == nil || resource.Resource.Name == "" {
		return nil
	}
	if deployConn, found := deployConns[resource.Resource.Name]; found {
		return deployConn
	}

	deploy := DeploymentConnectivity{Resource: *resource}
	deployConns[resource.Resource.Name] = &deploy
	return &deploy
}

func getDeployConnSelector(deployConn *DeploymentConnectivity) *metaV1.LabelSelector {
	selectorsMap := map[string]string{}
	for _, selector := range deployConn.Resource.Resource.Selectors {
		colonPos := strings.Index(selector, ":")
		if colonPos == -1 {
			continue
		}
		key := selector[:colonPos]
		value := selector[colonPos+1:]
		selectorsMap[key] = value
	}
	return &metaV1.LabelSelector{MatchLabels: selectorsMap}
}

func toNetpolPorts(ports []common.SvcNetworkAttr) []network.NetworkPolicyPort {
	var netpolPorts []network.NetworkPolicyPort
	for _, port := range ports {
		protocol := toCoreProtocol(port.Protocol)
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

func buildNetpolPerDeployment(deployConnectivity []*DeploymentConnectivity) []*network.NetworkPolicy {
	var netpols []*network.NetworkPolicy
	for _, deployConn := range deployConnectivity {
		if len(deployConn.egressConns) > 0 {
			deployConn.addEgressRule(nil, []network.NetworkPolicyPort{getDNSPort()})
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
				Ingress:     deployConn.ingressConns,
				Egress:      deployConn.egressConns,
				PolicyTypes: []network.PolicyType{network.PolicyTypeIngress, network.PolicyTypeEgress},
			},
		}
		netpols = append(netpols, &netpol)
	}
	return netpols
}

func getDNSPort() network.NetworkPolicyPort {
	udp := core.ProtocolUDP
	port53 := intstr.FromInt(dnsPort)
	return network.NetworkPolicyPort{
		Protocol: &udp,
		Port:     &port53,
	}
}

func synthNetpolList(connections []common.Connections) network.NetworkPolicyList {
	netpols := synthNetpols(connections)
	netpols2 := []network.NetworkPolicy{}
	for _, netpol := range netpols {
		netpols2 = append(netpols2, *netpol)
	}
	netpolList := network.NetworkPolicyList{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "NetworkPolicyList",
			APIVersion: "networking.k8s.io/v1",
		},
		Items: netpols2,
	}

	return netpolList
}
