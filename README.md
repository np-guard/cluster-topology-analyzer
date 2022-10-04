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

## Build the project
Make sure  you have golang 1.17+ on your platform

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
