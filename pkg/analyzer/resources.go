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
