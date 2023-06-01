package common

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type CfgMap struct {
	FullName string
	Data     map[string]string
}

type CfgMapKeyRef struct {
	Name string
	Key  string
}

type Resource struct {
	Resource struct {
		Name               string            `json:"name,omitempty"`
		Namespace          string            `json:"namespace,omitempty"`
		Labels             map[string]string `json:"labels,omitempty"`
		ServiceAccountName string            `json:"serviceaccountname,omitempty"`
		FilePath           string            `json:"filepath,omitempty"`
		Kind               string            `json:"kind,omitempty"`
		Image              struct {
			ID string `json:"id,omitempty"`
		} `json:"image"`
		NetworkAddrs     []string
		ConfigMapRefs    []string       `json:"-"`
		ConfigMapKeyRefs []CfgMapKeyRef `json:"-"`
		UsedPorts        []SvcNetworkAttr
	} `json:"resource,omitempty"`
}

type SvcNetworkAttr struct {
	Port       int                `json:"port,omitempty"`
	TargetPort intstr.IntOrString `json:"target_port,omitempty"`
	Protocol   corev1.Protocol    `json:"protocol,omitempty"`
}

type Service struct {
	Resource struct {
		Name             string             `json:"name,omitempty"`
		Namespace        string             `json:"namespace,omitempty"`
		Selectors        []string           `json:"selectors,omitempty"`
		Type             corev1.ServiceType `json:"type,omitempty"`
		FilePath         string             `json:"filepath,omitempty"`
		Kind             string             `json:"kind,omitempty"`
		Network          []SvcNetworkAttr   `json:"network,omitempty"`
		ExposeToCluster  bool               `json:"-"`
		ExposeExternally bool               `json:"-"`
	} `json:"resource,omitempty"`
}

type Connections struct {
	Source *Resource `json:"source,omitempty"`
	Target *Resource `json:"target"`
	Link   *Service  `json:"link"`
}

// A map from namespaces to a map of service names in each namespaces.
// For each service we also hold whether they should be exposed externally (true) or just globally inside the cluster (false)
type ServicesToExpose map[string]map[string]bool
