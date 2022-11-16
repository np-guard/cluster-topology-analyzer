# Shift-left Network Topology Analyzer

## About Topology Analyzer
This tool analyzes resource YAMLs of a Kubernetes-based application to extract the required connectivity between its workloads. It looks for network addresses that appear in workload specifications (e.g., in a Deployment's pod-spec environment) and correlates them to known and to predicted network addresses (e.g., expected Service URLs). Optionally, the tool can produce a list of NetworkPolicy resources that limit the workloads' connectivity to nothing but the detected required connectivity.

## Usage
```
$ ./bin/net-top -h
Usage of ./bin/net-top:
  -dirpath string
    	input directory path (required)
  -outputfile string
    	file path to store results
  -format string
        output format; must be either "json" or "yaml" (default "json")
  -netpols
        whether to synthesize NetworkPolicies to allow only the discovered connections
  -q    runs quietly, reports only severe errors and results
  -v    runs with more informative messages printed to log
```

## Algorithm
The underlying algorithm for identifying required connectivity works as follows.
1. Scan the given directory for all YAML files.
1. In each YAML file identify manifests for [workload resources](https://kubernetes.io/docs/concepts/workloads/controllers/), [Service resources](https://kubernetes.io/docs/concepts/services-networking/service/#service-resource) and [ConfigMap resources](https://kubernetes.io/docs/concepts/configuration/configmap/).
1. In each workload resource, inline references to ConfigMaps as if they were directly defined in the container's `envs` field.
1. For each target-workload in the list of workload resources:
    1. Identify all services whose selector matches target-workload
    1. For each such service:
        1. Compile a list of possible network addresses that can be used to access this service, e.g., `mysvc`, `mysvc.myns`, `mysvc.myns.svc.cluster.local`.
        1. Identify all workload resources with a container whose `envs` field contains a value from the list of possible network addresses, possibly with an additional port specifier.
        1. For each source-workload in the set of identified workloads:
            1. Add a connection from source-workload to target-workload to the list of identified connections. Add protocol and port information if available.

The algorithm for synthesizing NetworkPolicies that only allow the required connections and no other connection:
1. For each workload generate a NetworkPolicy resources as follows:
    - `metadata.namespace` is set to the workload's namespace (if specified)
    - `spec.podSelector` is set to the workload pod selector
    - `spec.policyTypes` is set to `["Ingress", "Egress"]`
    - `spec.ingress` contains one rule for each required connection in which the workload is the target workload
    - `spec.egress` contains one rule for each required connection in which the workload is the source workload. If such connections exist, also add a rule to allow egress to UDP port 53 (DNS).
1. For each **workload namespace** add a *default deny* NetworkPolicy as follows
    - `metadata.namespace` is set to the workload's namespace 
    - `spec.podSelector` is set to the empty selector (selects all pods in the namespace)
    - `spec.policyTypes` is set to `["Ingress", "Egress"]`
    - `spec.ingress` contains no rules (allows no ingress)
    - `spec.egress` contains no rules (allows no egress)

## Assumptions

1. All the relevant application resources (workloads, Services, ConfigMaps) are defined in YAML files under the given directory or its subdirectories
1. All YAML files can be applied to a Kubernetes cluster as-is using `kubectl apply -f` (i.e., no helm-style templating).
1. Every workload that needs to connect to a Service, will have the Service network address as the value of an environment variable. This can be specified directly in the containers `envs` (see example [here](tests/k8s_guestbook/frontend-deployment.yaml#L25:L28)) or via a ConfigMap (see examples [here](tests/onlineboutique/kubernetes-manifests.yaml#L105:L109) and [here](tests/onlineboutique/kubernetes-manifests.yaml#L269:L271)).
1. The network addresses of a given Service `<svc>` in Namespace `<ns>`, exposing port `<portNum>`, must match this pattern `(http(s)?://)?<svc>(.<ns>(.svc.cluster.local)?)?(:<portNum>)?`. Examples for legal network addresses are `wordpress-mysql:3306`, `redis-follower.redis.svc.cluster.local:6379`, `redis-leader.redis`, `http://rating-service`.

## Build the project
Make sure  you have golang 1.18+ on your platform

```shell
git clone git@github.com:np-guard/cluster-topology-analyzer.git
cd cluster-topology-analyzer
go mod download
make
```

## Run Examples

1. Clone a sample source code repository that you want to scan
```shell
git clone git@github.com:GoogleCloudPlatform/microservices-demo.git $HOME/microservices-demo
```

2. Point topology analyzer to this sample repo
```shell
./bin/net-top -dirpath $HOME/microservices-demo
```
3. The tool outputs a list of identified connections. Each connection is defined as a triplet: `<source, target, link>`. The list should look like this:
```json
[
    ...
    {
        "source": {
            "resource": {
                "name": "frontend",
                "selectors": [
                    "app:frontend"
                ],
                "labels": {
                    "app": "frontend"
                },
                "serviceaccountname": "default",
                "filepath": "/release/kubernetes-manifests.yaml",
                "kind": "Deployment",
                "image": {
                    "id": "gcr.io/google-samples/microservices-demo/frontend:v0.3.9"
                },
                "network": [
                    {
                        "container_url": 8080
                    }
                ],
                "Envs": [
                    "productcatalogservice:3550",
                    "currencyservice:7000",
                    "cartservice:7070",
                    "recommendationservice:8080",
                    "shippingservice:50051",
                    "checkoutservice:5050",
                    "adservice:9555"
                ],
                "UsedPorts": [
                    {
                        "port": 9555,
                        "target_port": 9555
                    }
                ]
            }
        },
        "target": {
            "resource": {
                "name": "adservice",
                "selectors": [
                    "app:adservice"
                ],
                "labels": {
                    "app": "adservice"
                },
                "serviceaccountname": "default",
                "filepath": "/release/kubernetes-manifests.yaml",
                "kind": "Deployment",
                "image": {
                    "id": "gcr.io/google-samples/microservices-demo/adservice:v0.3.9"
                },
                "network": [
                    {
                        "container_url": 9555
                    }
                ],
                "Envs": null,
                "UsedPorts": null
            }
        },
        "link": {
            "resource": {
                "name": "adservice",
                "selectors": [
                    "app:adservice"
                ],
                "type": "ClusterIP",
                "filepath": "/release/kubernetes-manifests.yaml",
                "kind": "Service",
                "network": [
                    {
                        "port": 9555,
                        "target_port": 9555
                    }
                ]
            }
        }
    }
]
```
4. Produce NetworkPolicies for this sample repo to only allow detected connectivity, and store them in `netpols.json` as a single NetworkPolicyList resource. Run quietly.
```shell
./bin/net-top -dirpath $HOME/microservices-demo -netpols -outputfile netpols.json -q
```

## Golang API
The functionality of this tool can be consumed via a [Golang package API](https://pkg.go.dev/github.com/np-guard/cluster-topology-analyzer/pkg/controller). The relevant package to import is `github.com/np-guard/cluster-topology-analyzer/pkg/controller`.

Main functionality is encapsulated under the `PoliciesSynthesizer`, which exposes two methods:
* `func (ps *PoliciesSynthesizer) ConnectionsFromFolderPath(dirPath string) ([]*common.Connections, error)` - getting a slice of Connection objects, each representing a required connection in the scan application.
* `func (ps *PoliciesSynthesizer) PoliciesFromFolderPath(dirPath string) ([]*networking.NetworkPolicy, error)` - getting a slice of K8s NetworkPolicy objects that limit the allowed connectivity to only the required connections.

The example code below extracts required connections from the K8s manifests in the `/tmp/k8s_manifests` directory, and outputs appropriate K8s NetworkPolicies to standard output.
```golang
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
)

func main() {
	logger := controller.NewDefaultLogger()
	synth := controller.NewPoliciesSynthesizer(controller.WithLogger(logger))

	netpols, err := synth.PoliciesFromFolderPath("/tmp/k8s_manifests")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error synthesizing policies: %v\n", err)
		os.Exit(1)
	}
	buf, _ := json.MarshalIndent(netpols, "", "    ")
	fmt.Printf("%v\n", string(buf))
}
```
