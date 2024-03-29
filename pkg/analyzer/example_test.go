/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer_test

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/np-guard/cluster-topology-analyzer/v2/pkg/analyzer"
)

func ExamplePoliciesSynthesizer() {
	logger := analyzer.NewDefaultLogger()
	synth := analyzer.NewPoliciesSynthesizer(analyzer.WithLogger(logger))

	netpols, err := synth.PoliciesFromFolderPath("../../tests/k8s_wordpress_example")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error synthesizing policies: %v\n", err)
		os.Exit(1)
	}
	buf, _ := json.MarshalIndent(netpols, "", "    ")
	fmt.Printf("%v\n", string(buf))
	// Output:
	// [
	//     {
	//         "kind": "NetworkPolicy",
	//         "apiVersion": "networking.k8s.io/v1",
	//         "metadata": {
	//             "name": "wordpress-netpol",
	//             "creationTimestamp": null
	//         },
	//         "spec": {
	//             "podSelector": {
	//                 "matchLabels": {
	//                     "app": "wordpress",
	//                     "tier": "frontend"
	//                 }
	//             },
	//             "ingress": [
	//                 {
	//                     "ports": [
	//                         {
	//                             "protocol": "TCP",
	//                             "port": 80
	//                         }
	//                     ]
	//                 }
	//             ],
	//             "egress": [
	//                 {
	//                     "ports": [
	//                         {
	//                             "protocol": "TCP",
	//                             "port": 3306
	//                         }
	//                     ],
	//                     "to": [
	//                         {
	//                             "podSelector": {
	//                                 "matchLabels": {
	//                                     "app": "wordpress",
	//                                     "tier": "mysql"
	//                                 }
	//                             }
	//                         }
	//                     ]
	//                 },
	//                 {
	//                     "ports": [
	//                         {
	//                             "protocol": "UDP",
	//                             "port": 53
	//                         }
	//                     ],
	//                     "to": [
	//                         {
	//                             "namespaceSelector": {}
	//                         }
	//                     ]
	//                 }
	//             ],
	//             "policyTypes": [
	//                 "Ingress",
	//                 "Egress"
	//             ]
	//         }
	//     },
	//     {
	//         "kind": "NetworkPolicy",
	//         "apiVersion": "networking.k8s.io/v1",
	//         "metadata": {
	//             "name": "wordpress-mysql-netpol",
	//             "creationTimestamp": null
	//         },
	//         "spec": {
	//             "podSelector": {
	//                 "matchLabels": {
	//                     "app": "wordpress",
	//                     "tier": "mysql"
	//                 }
	//             },
	//             "ingress": [
	//                 {
	//                     "ports": [
	//                         {
	//                             "protocol": "TCP",
	//                             "port": 3306
	//                         }
	//                     ],
	//                     "from": [
	//                         {
	//                             "podSelector": {
	//                                 "matchLabels": {
	//                                     "app": "wordpress",
	//                                     "tier": "frontend"
	//                                 }
	//                             }
	//                         }
	//                     ]
	//                 }
	//             ],
	//             "policyTypes": [
	//                 "Ingress",
	//                 "Egress"
	//             ]
	//         }
	//     },
	//     {
	//         "kind": "NetworkPolicy",
	//         "apiVersion": "networking.k8s.io/v1",
	//         "metadata": {
	//             "name": "default-deny-in-namespace",
	//             "creationTimestamp": null
	//         },
	//         "spec": {
	//             "podSelector": {},
	//             "policyTypes": [
	//                 "Ingress",
	//                 "Egress"
	//             ]
	//         }
	//     }
	// ]
}
