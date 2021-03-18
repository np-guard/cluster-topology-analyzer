package analyzer

import (
	"bytes"
	"fmt"
	"strings"

	"github.ibm.com/gitsecure-net-top/pkg/common"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
)

//ScanK8sDeployObject :
func ScanK8sDeployObject(kind string, objDataBuf []byte) (common.Resource, error) {
	var podSpecv1 v1.PodTemplateSpec
	var resourceCtx common.Resource
	// var replicaCount int32
	switch kind {
	case "Pod":
		zap.S().Info("evaluating pod")
		// obj := parser.ParsePod(bytes.NewReader(dataBuf))
		// podSpecv1 = obj.Spec
	case "ReplicaSet":
		obj := ParseReplicaSet(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		// resourceCtx.Namespace = obj.GetNamespace()
		podSpecv1 = obj.Spec.Template
		// resourceCtx.Resource.ReplicaCount = int(*obj.Spec.Replicas)
	case "ReplicationController":
		obj := ParseReplicationController(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		// for k, v := range obj.Spec.Selector.MatchLabels {
		// 	resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		// }
		podSpecv1 = *obj.Spec.Template
		// resourceCtx.Resource.ReplicaCount = int(*obj.Spec.Replicas)
	case "Deployment":
		obj := ParseDeployment(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		podSpecv1 = obj.Spec.Template
		// resourceCtx.Resource.ReplicaCount = int(*obj.Spec.Replicas)
	case "DaemonSet":
		obj := ParseDaemonSet(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		podSpecv1 = obj.Spec.Template
	case "StatefulSet":
		obj := ParseStatefulSet(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		podSpecv1 = obj.Spec.Template
	case "Job":
		obj := ParseJob(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Name = obj.GetName()
		resourceCtx.Resource.Namespace = obj.GetNamespace()
		resourceCtx.Resource.Kind = kind
		for k, v := range obj.Spec.Selector.MatchLabels {
			resourceCtx.Resource.Selectors = append(resourceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		podSpecv1 = obj.Spec.Template
	default:
		return resourceCtx, fmt.Errorf("unsupported object type: `%s`", kind)
	}

	parseDeployResource(podSpecv1, &resourceCtx)
	return resourceCtx, nil
}

//ScanK8sServiceObject :
func ScanK8sServiceObject(kind string, objDataBuf []byte) (common.Service, error) {
	var svcSpecv1 v1.ServiceSpec
	var serviceCtx common.Service
	switch kind {
	case "Service":
		svcObj := ParseService(bytes.NewReader(objDataBuf))
		serviceCtx.Resource.Name = svcObj.GetName()
		serviceCtx.Resource.Namespace = svcObj.Namespace
		serviceCtx.Resource.Kind = kind
		serviceCtx.Resource.Type = string(svcObj.Spec.Type)
		for k, v := range svcObj.Spec.Selector {
			serviceCtx.Resource.Selectors = append(serviceCtx.Resource.Selectors, fmt.Sprintf("%s:%s", k, v))
		}
		// serviceCtx.Resource.Selectors = svcObj.GetLabels()
		svcSpecv1 = svcObj.Spec
	default:
		return serviceCtx, fmt.Errorf("unsupported object type: `%s`", kind)
	}
	parseServiceResource(svcSpecv1, &serviceCtx)

	return serviceCtx, nil
}

func parseDeployResource(podSpec v1.PodTemplateSpec, resourceCtx *common.Resource) error {
	for _, container := range podSpec.Spec.Containers {
		resourceCtx.Resource.Image.ID = container.Image
		for _, p := range container.Ports {
			n := common.NetworkAttr{}
			n.ContainerPort = int(p.ContainerPort)
			n.HostPort = int(p.HostPort)
			n.Protocol = string(p.Protocol)
			resourceCtx.Resource.Network = append(resourceCtx.Resource.Network, n)
		}
		for _, e := range container.Env {
			if strings.Contains(e.Value, ":") {
				resourceCtx.Resource.Envs = append(resourceCtx.Resource.Envs, e.Value)
			}
		}
	}
	return nil
}

func parseServiceResource(svcSpec v1.ServiceSpec, serviceCtx *common.Service) error {
	for _, p := range svcSpec.Ports {
		n := common.SvcNetworkAttr{}
		n.Port = int(p.Port)
		n.TargetPort = int(p.TargetPort.IntVal)
		n.Protocol = string(p.Protocol)
		serviceCtx.Resource.Network = append(serviceCtx.Resource.Network, n)
	}
	return nil
}
