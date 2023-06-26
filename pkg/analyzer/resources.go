/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"bytes"

	"k8s.io/apimachinery/pkg/util/yaml"
)

const yamlParseBufferSize = 200

func parseResource[T interface{}](objDataBuf []byte) *T {
	reader := bytes.NewReader(objDataBuf)
	if reader == nil {
		return nil
	}
	var rc T
	err := yaml.NewYAMLOrJSONDecoder(reader, yamlParseBufferSize).Decode(&rc)
	if err != nil {
		return nil
	}
	return &rc
}
