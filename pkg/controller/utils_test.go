package controller

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func Test_splitByYamlDocuments(t *testing.T) {
	currentDir, _ := os.Getwd()
	tests := []struct {
		name        string
		yamlFile    string
		mode        ErrMode
		wantNumDocs int
		wantErr     bool
	}{
		{
			name:        "kubernetes-manifests.yaml should have 25 documents",
			yamlFile:    filepath.Join(currentDir, "..", "..", "tests", "onlineboutique", "kubernetes-manifests.yaml"),
			mode:        Strict,
			wantNumDocs: 25,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filebuf, err := os.ReadFile(tt.yamlFile)
			assert.NoError(t, err)
			got, err := splitByYamlDocuments(filebuf, tt.mode)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantNumDocs, len(got))
		})
	}
}
