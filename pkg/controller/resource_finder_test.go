package controller

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRelevantK8sResourcesBadYamlDocument(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	resFinder := resourceFinder{logger: NewDefaultLogger(), stopOn1stErr: false, walkFn: filepath.WalkDir}
	objs, errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDoc := &MalformedYamlDocError{}
	require.True(t, errors.As(errs[0].Error(), &badDoc))

	docID, err := errs[0].DocumentID()
	require.Equal(t, 6, docID)
	require.Nil(t, err)

	require.Len(t, objs, 1)
	require.Len(t, objs[0].rawK8sResources, 6)
}

func TestGetRelevantK8sResourcesBadYamlDocumentFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	resFinder := resourceFinder{logger: NewDefaultLogger(), stopOn1stErr: true, walkFn: filepath.WalkDir}
	objs, errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDoc := &MalformedYamlDocError{}
	require.True(t, errors.As(errs[0].Error(), &badDoc))

	docID, err := errs[0].DocumentID()
	require.Equal(t, 6, docID)
	require.Nil(t, err)

	require.Empty(t, objs)
}

func TestGetRelevantK8sResourcesNoK8sResource(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "not_a_k8s_resource.yaml")
	resFinder := resourceFinder{logger: NewDefaultLogger(), stopOn1stErr: false, walkFn: filepath.WalkDir}
	objs, errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	notK8sRes := &NotK8sResourceError{}
	require.True(t, errors.As(errs[0].Error(), &notK8sRes))
	require.Len(t, objs, 1)
	require.Len(t, objs[0].rawK8sResources, 1)
}

func TestGetRelevantK8sResourcesNoYAMLs(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir2")
	resFinder := resourceFinder{logger: NewDefaultLogger(), stopOn1stErr: false, walkFn: filepath.WalkDir}
	objs, errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	noYamls := &NoYamlsFoundError{}
	require.True(t, errors.As(errs[0].Error(), &noYamls))
	require.Empty(t, objs)
}

func TestGetRelevantK8sResourcesBadDir(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	resFinder := resourceFinder{logger: NewDefaultLogger(), stopOn1stErr: false, walkFn: filepath.WalkDir}
	objs, errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDir := &FailedAccessingDirError{}
	require.True(t, errors.As(errs[0].Error(), &badDir))
	require.Empty(t, objs)
}

func TestGetRelevantK8sResourcesBadDirFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	resFinder := resourceFinder{logger: NewDefaultLogger(), stopOn1stErr: true, walkFn: filepath.WalkDir}
	objs, errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDir := &FailedAccessingDirError{}
	require.True(t, errors.As(errs[0].Error(), &badDir))
	require.Empty(t, objs)
}

func TestSearchForManifests(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	resFinder := resourceFinder{logger: NewDefaultLogger(), stopOn1stErr: false, walkFn: filepath.WalkDir}
	yamlFiles, errs := resFinder.searchForManifests(dirPath)
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
	resFinder := resourceFinder{logger: NewDefaultLogger(), stopOn1stErr: false, walkFn: nonRecursiveWalk}
	yamlFiles, errs := resFinder.searchForManifests(dirPath)
	require.Empty(t, errs)
	require.Len(t, yamlFiles, 4)
}
