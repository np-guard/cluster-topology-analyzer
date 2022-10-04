package controller

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetK8sDeploymentResourcesBadYamlDocument(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	objs, errs := getK8sDeploymentResources(dirPath, false)
	require.Len(t, errs, 1)

	docID, err := errs[0].DocumentID()
	require.Equal(t, 6, docID)
	require.Nil(t, err)

	require.Len(t, objs, 1)
	require.Len(t, objs[0].DeployObjects, 6)
}

func TestGetK8sDeploymentResourcesBadYamlDocumentFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	objs, errs := getK8sDeploymentResources(dirPath, true)
	require.Len(t, errs, 1)

	docID, err := errs[0].DocumentID()
	require.Equal(t, 6, docID)
	require.Nil(t, err)

	require.Empty(t, objs)
}

func TestGetK8sDeploymentResourcesNoK8sResource(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "not_a_k8s_resource.yaml")
	objs, errs := getK8sDeploymentResources(dirPath, false)
	require.Len(t, errs, 1)
	require.Len(t, objs, 1)
	require.Len(t, objs[0].DeployObjects, 1)
}

func TestGetK8sDeploymentResourcesNoYAMLs(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir2")
	objs, errs := getK8sDeploymentResources(dirPath, false)
	require.Len(t, errs, 1)
	require.Empty(t, objs)
}

func TestGetK8sDeploymentResourcesBadDir(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	objs, errs := getK8sDeploymentResources(dirPath, false)
	require.Len(t, errs, 1)
	require.Empty(t, objs)
}

func TestGetK8sDeploymentResourcesBadDirFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	objs, errs := getK8sDeploymentResources(dirPath, true)
	require.Len(t, errs, 1)
	require.Empty(t, objs)
}

func TestSearchDeploymentManifests(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	yamlFiles, errs := searchDeploymentManifests(dirPath, false)
	require.Empty(t, errs)
	require.Len(t, yamlFiles, 5)
}
