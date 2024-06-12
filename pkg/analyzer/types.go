/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type cfgMap struct {
	FullName string
	Data     map[string]string
}

type cfgMapKeyRef struct {
	Name string
	Key  string
}

// Resource is an abstraction of a k8s workload resource (e.g., pod, deployment).
// It also stores additional information that is later being used in the analysis
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
		ConfigMapKeyRefs []cfgMapKeyRef `json:"-"`
		UsedPorts        []SvcNetworkAttr
	} `json:"resource,omitempty"`
}

func (r1 *Resource) equals(r2 *Resource) bool {
	return r1.Resource.Name == r2.Resource.Name &&
		r1.Resource.Namespace == r2.Resource.Namespace &&
		r1.Resource.Kind == r2.Resource.Kind
}

// SvcNetworkAttr is used to store port information
type SvcNetworkAttr struct {
	Port       int                `json:"port,omitempty"`
	TargetPort intstr.IntOrString `json:"target_port,omitempty"`
	Protocol   corev1.Protocol    `json:"protocol,omitempty"`
}

// Service is used to store information about a K8s Service
type Service struct {
	Resource struct {
		Name             string             `json:"name,omitempty"`
		Namespace        string             `json:"namespace,omitempty"`
		Selectors        []string           `json:"selectors,omitempty"`
		Type             corev1.ServiceType `json:"type,omitempty"`
		FilePath         string             `json:"filepath,omitempty"`
		Kind             string             `json:"kind,omitempty"`
		Network          []SvcNetworkAttr   `json:"network,omitempty"`
		exposeToNS       []string
		ExposeToCluster  bool `json:"-"`
		ExposeExternally bool `json:"-"`
	} `json:"resource,omitempty"`
}

// Connections represents a connection from a source workload to a target workload using via a service.
type Connections struct {
	Source *Resource `json:"source,omitempty"`
	Target *Resource `json:"target"`
	Link   *Service  `json:"link"`
}

// A map from namespaces to a map of service names in each namespaces.
// For each service we also hold whether they should be exposed externally (true) or just globally inside the cluster (false)
type servicesToExpose map[string]map[string]bool
