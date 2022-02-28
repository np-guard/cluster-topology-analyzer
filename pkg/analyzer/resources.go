package analyzer

import (
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
)

// ParsePod parses replicationController
func ParsePod(r io.Reader) *v1.Pod {
	if r == nil {
		return nil
	}
	rc := v1.Pod{}
	err := yaml.NewYAMLOrJSONDecoder(r, 200).Decode(&rc)
	if err != nil {
		return nil
	}
	return &rc
}

// ParseDeployment parses deployment
func ParseDeployment(r io.Reader) *appsv1.Deployment {
	if r == nil {
		return nil
	}
	rc := appsv1.Deployment{}
	err := yaml.NewYAMLOrJSONDecoder(r, 100).Decode(&rc)
	if err != nil {
		return nil
	}
	return &rc
}

// ParseReplicaSet parses replicaset
func ParseReplicaSet(r io.Reader) *appsv1.ReplicaSet {
	if r == nil {
		return nil
	}
	rc := appsv1.ReplicaSet{}
	err := yaml.NewYAMLOrJSONDecoder(r, 200).Decode(&rc)
	if err != nil {
		return nil
	}
	return &rc
}

// ParseReplicationController parses replicationController
func ParseReplicationController(r io.Reader) *v1.ReplicationController {
	if r == nil {
		return nil
	}
	rc := v1.ReplicationController{}
	err := yaml.NewYAMLOrJSONDecoder(r, 200).Decode(&rc)
	if err != nil {
		return nil
	}

	return &rc
}

// ParseDaemonSet parses replicationController
func ParseDaemonSet(r io.Reader) *appsv1.DaemonSet {
	if r == nil {
		return nil
	}
	rc := appsv1.DaemonSet{}
	err := yaml.NewYAMLOrJSONDecoder(r, 200).Decode(&rc)
	if err != nil {
		return nil
	}

	return &rc
}

// ParseStatefulSet parses replicationController
func ParseStatefulSet(r io.Reader) *appsv1.StatefulSet {
	if r == nil {
		return nil
	}
	rc := appsv1.StatefulSet{}
	err := yaml.NewYAMLOrJSONDecoder(r, 200).Decode(&rc)
	if err != nil {
		return nil
	}

	return &rc
}

// ParseJob parses replicationController
func ParseJob(r io.Reader) *batchv1.Job {
	if r == nil {
		return nil
	}
	rc := batchv1.Job{}
	err := yaml.NewYAMLOrJSONDecoder(r, 200).Decode(&rc)
	if err != nil {
		return nil
	}

	return &rc
}

// ParseService parses replicationController
func ParseService(r io.Reader) *v1.Service {
	if r == nil {
		return nil
	}
	rc := v1.Service{}
	err := yaml.NewYAMLOrJSONDecoder(r, 200).Decode(&rc)
	if err != nil {
		return nil
	}
	return &rc
}

// ParseServiceAccount parses replicationController
func ParseServiceAccount(r io.Reader) *v1.ServiceAccount {
	if r == nil {
		return nil
	}
	rc := v1.ServiceAccount{}
	err := yaml.NewYAMLOrJSONDecoder(r, 200).Decode(&rc)
	if err != nil {
		return nil
	}
	return &rc
}

// ParseConfigMap parses ConfigMap
func ParseConfigMap(r io.Reader) *v1.ConfigMap {
	if r == nil {
		return nil
	}
	rc := v1.ConfigMap{}
	err := yaml.NewYAMLOrJSONDecoder(r, 200).Decode(&rc)
	if err != nil {
		return nil
	}
	return &rc
}
