/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"
)

const yamlParseBufferSize = 200

func parseResource[T interface{}](r io.Reader) *T {
	if r == nil {
		return nil
	}
	var rc T
	err := yaml.NewYAMLOrJSONDecoder(r, yamlParseBufferSize).Decode(&rc)
	if err != nil {
		return nil
	}
	return &rc
}
