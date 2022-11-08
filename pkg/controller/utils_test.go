package controller

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRelevantK8sResourcesBadYamlDocument(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	objs, errs := getRelevantK8sResources(dirPath, false, filepath.WalkDir)
	require.Len(t, errs, 1)

	docID, err := errs[0].DocumentID()
	require.Equal(t, 6, docID)
	require.Nil(t, err)

	require.Len(t, objs, 1)
	require.Len(t, objs[0].DeployObjects, 6)
}

func TestGetRelevantK8sResourcesBadYamlDocumentFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	objs, errs := getRelevantK8sResources(dirPath, true, filepath.WalkDir)
	require.Len(t, errs, 1)

	docID, err := errs[0].DocumentID()
	require.Equal(t, 6, docID)
	require.Nil(t, err)

	require.Empty(t, objs)
}

func TestGetRelevantK8sResourcesNoK8sResource(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "not_a_k8s_resource.yaml")
	objs, errs := getRelevantK8sResources(dirPath, false, filepath.WalkDir)
	require.Len(t, errs, 1)
	require.Len(t, objs, 1)
	require.Len(t, objs[0].DeployObjects, 1)
}

func TestGetRelevantK8sResourcesNoYAMLs(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir2")
	objs, errs := getRelevantK8sResources(dirPath, false, filepath.WalkDir)
	require.Len(t, errs, 1)
	require.Empty(t, objs)
}

func TestGetRelevantK8sResourcesBadDir(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	objs, errs := getRelevantK8sResources(dirPath, false, filepath.WalkDir)
	require.Len(t, errs, 1)
	require.Empty(t, objs)
}

func TestGetRelevantK8sResourcesBadDirFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	objs, errs := getRelevantK8sResources(dirPath, true, filepath.WalkDir)
	require.Len(t, errs, 1)
	require.Empty(t, objs)
}

func TestSearchForManifests(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	yamlFiles, errs := searchForManifests(dirPath, false, filepath.WalkDir)
	require.Empty(t, errs)
	require.Len(t, yamlFiles, 5)
}

func nonRecursiveWalk(root string, fn fs.WalkDirFunc) error {
	err := filepath.WalkDir(root, func(path string, f os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		if f == nil || path != root && f.IsDir() {
			return filepath.SkipDir
		}
		return fn(path, f, err)
	})
	return err
}

func TestSearchForManifestsNonRecursiveWalk(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	yamlFiles, errs := searchForManifests(dirPath, false, nonRecursiveWalk)
	require.Empty(t, errs)
	require.Len(t, yamlFiles, 4)
}
