package controller

import (
	"reflect"
	"sort"

	core "k8s.io/api/core/v1"
	network "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

const (
	dnsPort           = 53
	networkAPIVersion = "networking.k8s.io/v1"
)

type deploymentConnectivity struct {
	common.Resource
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

func synthNetpols(connections []*common.Connections) []*network.NetworkPolicy {
	deployConnectivity := determineConnectivityPerDeployment(connections)
	netpols := buildNetpolPerDeployment(deployConnectivity)
	return netpols
}

func determineConnectivityPerDeployment(connections []*common.Connections) []*deploymentConnectivity {
	deploysConnectivity := map[string]*deploymentConnectivity{}
	for _, conn := range connections {
		srcDeploy := findOrAddDeploymentConn(conn.Source, deploysConnectivity)
		dstDeploy := findOrAddDeploymentConn(conn.Target, deploysConnectivity)
		targetPorts := toNetpolPorts(conn.Link.Resource.Network)
		if conn.Source != nil && len(conn.Source.Resource.UsedPorts) > 0 {
			targetPorts = toNetpolPorts(conn.Source.Resource.UsedPorts)
		}

		if srcDeploy != nil {
			netpolPeer := getNetpolPeer(srcDeploy, dstDeploy)
			srcDeploy.addEgressRule([]network.NetworkPolicyPeer{netpolPeer}, targetPorts)
		}

		if conn.Link.Resource.Type == "LoadBalancer" || conn.Link.Resource.Type == "NodePort" {
			dstDeploy.addIngressRule([]network.NetworkPolicyPeer{}, targetPorts) // in these cases we want to allow traffic from all sources
		} else if conn.Source != nil {
			netpolPeer := getNetpolPeer(dstDeploy, srcDeploy)
			dstDeploy.addIngressRule([]network.NetworkPolicyPeer{netpolPeer}, targetPorts) // allow traffic only from this specific source
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

func findOrAddDeploymentConn(resource *common.Resource, deployConns map[string]*deploymentConnectivity) *deploymentConnectivity {
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

func getNetpolPeer(netpolDeploy, otherDeploy *deploymentConnectivity) network.NetworkPolicyPeer {
	netpolPeer := network.NetworkPolicyPeer{PodSelector: getDeployConnSelector(otherDeploy)}
	if netpolDeploy.Resource.Resource.Namespace != otherDeploy.Resource.Resource.Namespace {
		if otherDeploy.Resource.Resource.Namespace != "" {
			netpolPeer.NamespaceSelector = &metaV1.LabelSelector{
				MatchLabels: map[string]string{"kubernetes.io/metadata.name": otherDeploy.Resource.Resource.Namespace},
			}
		} // if otherDeploy has no namespace specified, we assume it is in the same namespace as the netpolDeploy
	}
	return netpolPeer
}

func getDeployConnSelector(deployConn *deploymentConnectivity) *metaV1.LabelSelector {
	return &metaV1.LabelSelector{MatchLabels: deployConn.Resource.Resource.Labels}
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

func buildNetpolPerDeployment(deployConnectivity []*deploymentConnectivity) []*network.NetworkPolicy {
	var netpols []*network.NetworkPolicy
	for _, deployConn := range deployConnectivity {
		if len(deployConn.egressConns) > 0 {
			deployConn.addEgressRule(nil, []network.NetworkPolicyPort{getDNSPort()})
		}
		netpol := network.NetworkPolicy{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "NetworkPolicy",
				APIVersion: networkAPIVersion,
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

func NetpolListFromNetpolSlice(netpols []*network.NetworkPolicy) network.NetworkPolicyList {
	netpols2 := []network.NetworkPolicy{}
	for _, netpol := range netpols {
		netpols2 = append(netpols2, *netpol)
	}
	netpolList := network.NetworkPolicyList{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "NetworkPolicyList",
			APIVersion: networkAPIVersion,
		},
		Items: netpols2,
	}

	return netpolList
}
