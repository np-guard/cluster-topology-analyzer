# Shift-left Network Topology Analyzer

## About Topology Analyzer
This tool analyzes resource YAMLs of a Kubernetes-based application to extract the required connectivity between its workloads. It looks for network addresses that appear in workload specifications (e.g., in a Deployment's pod-spec environment) and correlates them to known and to predicted network addresses (e.g., expected Service URLs). Optionally, the tool can produce a list of NetworkPolicy resources that limit the workloads' connectivity to nothing but the detected required connectivity.

## Usage
```
$ ./bin/net-top -h
Usage of ./bin/net-top:
  -dirpath string
    	input directory path (required, can be specified multiple times with different directories)
  -outputfile string
    	file path to store results
  -format string
        output format; must be either "json" or "yaml" (default "json")
  -netpols
        whether to synthesize NetworkPolicies to allow only the discovered connections
  -dnsport int
        specify DNS port to be used in egress rules of synthesized NetworkPolicies (default 53)
  -q    runs quietly, reports only severe errors and results
  -v    runs with more informative messages printed to log
```

## Algorithm
The underlying algorithm for identifying required connectivity works as follows.
1. Scan the given directories for all YAML files.
1. In each YAML file identify manifests for [workload resources](https://kubernetes.io/docs/concepts/workloads/controllers/), [Service resources](https://kubernetes.io/docs/concepts/services-networking/service/#service-resource) and [ConfigMap resources](https://kubernetes.io/docs/concepts/configuration/configmap/), [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress) and [Route](https://docs.openshift.com/container-platform/latest/networking/routes/route-configuration.html).
1. In each workload resource, identify configuration values that might represent network addresses. This includes strings in containers' `envs`, `args` and `command` fields, as well as references to data in ConfigMaps.
1. For each target-workload in the list of workload resources:
    1. Identify all services whose selector matches target-workload
    1. For each such service:
        1. Compile a list of possible network addresses that can be used to access this service, e.g., `mysvc`, `mysvc.myns`, `mysvc.myns.svc.cluster.local`.
        1. Identify all workload resources with a configuration value that matches a value from the list of possible network addresses, possibly with an additional port specifier.
        1. For each source-workload in the set of identified workloads:
            1. Add a connection from source-workload to target-workload to the list of identified connections. Add protocol and port information if available.

The algorithm for synthesizing NetworkPolicies that only allow the required connections and no other connection:
1. For each workload generate a NetworkPolicy resources as follows:
    - `metadata.namespace` is set to the workload's namespace (if specified)
    - `spec.podSelector` is set to the workload pod selector
    - `spec.policyTypes` is set to `["Ingress", "Egress"]`
    - `spec.ingress` contains one rule for each required connection in which the workload is the target workload. If the Service exposing this workload is of type `LoadBalancer` or `NodePort`, allow ingress from any source. If the service exposing this workload is pointed by an Ingress resource or by a Route resource, allow ingress from any source within the cluster.
    - `spec.egress` contains one rule for each required connection in which the workload is the source workload. If such connections exist, also add a rule to allow egress to UDP port 53 (DNS).
1. For each **workload namespace** add a *default deny* NetworkPolicy as follows
    - `metadata.namespace` is set to the workload's namespace 
    - `spec.podSelector` is set to the empty selector (selects all pods in the namespace)
    - `spec.policyTypes` is set to `["Ingress", "Egress"]`
    - `spec.ingress` contains no rules (allows no ingress)
    - `spec.egress` contains no rules (allows no egress)

## Assumptions

1. All the relevant application resources (workloads, Services, ConfigMaps) are defined in YAML files under the given directories or their subdirectories
1. All YAML files can be applied to a Kubernetes cluster as-is using `kubectl apply -f` (i.e., no helm-style templating).
1. Every workload that needs to connect to a Service, will somehow specify the network address of this Service in its manifest. This can be specified directly in the containers `envs` (see example [here](tests/k8s_guestbook/frontend-deployment.yaml#L25:L28)), or via a ConfigMap (see examples [here](tests/onlineboutique/kubernetes-manifests.yaml#L110:L114) and [here](tests/onlineboutique/kubernetes-manifests.yaml#L270:L272)), or using command-line arguments.
1. The network addresses of a given Service `<svc>` in Namespace `<ns>`, exposing port `<portNum>`, must match this pattern `(http(s)?://)?<svc>(.<ns>(.svc.cluster.local)?)?(:<portNum>)?`. Examples for legal network addresses are `wordpress-mysql:3306`, `redis-follower.redis.svc.cluster.local:6379`, `redis-leader.redis`, `http://rating-service`.

## Build the project
Make sure you have golang 1.20+ on your platform

```shell
git clone git@github.com:np-guard/cluster-topology-analyzer.git
cd cluster-topology-analyzer
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
                "labels": {
                    "app": "frontend"
                },
                "serviceaccountname": "default",
                "filepath": "kubernetes-manifests.yaml",
                "kind": "Deployment",
                "image": {
                    "id": "gcr.io/google-samples/microservices-demo/frontend:v0.2.3"
                },
                "NetworkAddrs": [
                    "productcatalogservice:3550",
                    "currencyservice:7000",
                    "cartservice:7070",
                    "recommendationservice:8080",
                    "checkoutservice:5050",
                    "adservice:9555",
                    "shippingservice:50051"
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
                "labels": {
                    "app": "adservice"
                },
                "serviceaccountname": "default",
                "filepath": "kubernetes-manifests.yaml",
                "kind": "Deployment",
                "image": {
                    "id": "gcr.io/google-samples/microservices-demo/adservice:v0.2.3"
                },
                "NetworkAddrs": null,
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
                "filepath": "kubernetes-manifests.yaml",
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
The functionality of this tool can be consumed via a [Golang package API](https://pkg.go.dev/github.com/np-guard/cluster-topology-analyzer/pkg/analyzer). The relevant package to import is `github.com/np-guard/cluster-topology-analyzer/pkg/analyzer`.

Main functionality is encapsulated under the `PoliciesSynthesizer`, which exposes six methods:
* `func (ps *PoliciesSynthesizer) ConnectionsFromFolderPath(dirPath string) ([]*common.Connections, error)` - getting a slice of Connection objects, each representing a required connection in the scanned application.
* `func (ps *PoliciesSynthesizer) ConnectionsFromFolderPaths(dirPaths []string) ([]*common.Connections, error)` - same as `ConnectionsFromFolderPath()` but allows specifying multiple directories to scan.
* `func (ps *PoliciesSynthesizer) ConnectionsFromInfos(infos []*resource.Info) ([]*Connections, error)` - same as `ConnectionsFromFolderPath()` but analyzing the K8s resources in a slice of `Info` objects rather than scanning a file-system directory for manifest files.
* `func (ps *PoliciesSynthesizer) PoliciesFromFolderPath(dirPath string) ([]*networking.NetworkPolicy, error)` - getting a slice of K8s NetworkPolicy objects that limit the allowed connectivity to only the required connections.
* `func (ps *PoliciesSynthesizer) PoliciesFromFolderPaths(dirPaths []string) ([]*networking.NetworkPolicy, error)` - same as `PoliciesFromFolderPath()` but allows specifying multiple directories to scan.
* `func (ps *PoliciesSynthesizer) PoliciesFromInfos(infos []*resource.Info) ([]*networking.NetworkPolicy, error)` - same as `PoliciesFromFolderPath()` but analyzing the K8s resources in a slice of `Info` objects rather than scanning a file-system directory for manifest files.

The example code below extracts required connections from the K8s manifests in the `/tmp/k8s_manifests` directory, and outputs appropriate K8s NetworkPolicies to standard output.
```golang
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/np-guard/cluster-topology-analyzer/v2/pkg/analyzer"
)

func main() {
	logger := analyzer.NewDefaultLogger()
	synth := analyzer.NewPoliciesSynthesizer(analyzer.WithLogger(logger))

	netpols, err := synth.PoliciesFromFolderPath("/tmp/k8s_manifests")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error synthesizing policies: %v\n", err)
		os.Exit(1)
	}
	buf, _ := json.MarshalIndent(netpols, "", "    ")
	fmt.Printf("%v\n", string(buf))
}
```
