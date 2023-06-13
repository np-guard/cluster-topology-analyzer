/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

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
	resFinder := newResourceFinder(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDoc := &MalformedYamlDocError{}
	require.True(t, errors.As(errs[0].Error(), &badDoc))

	docID, err := errs[0].DocumentID()
	require.Equal(t, 6, docID)
	require.Nil(t, err)

	require.Len(t, resFinder.workloads, 3)
	require.Len(t, resFinder.services, 3)
	require.Empty(t, resFinder.configmaps)
}

func TestGetRelevantK8sResourcesBadYamlDocumentFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	resFinder := newResourceFinder(NewDefaultLogger(), true, filepath.WalkDir)
	errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDoc := &MalformedYamlDocError{}
	require.True(t, errors.As(errs[0].Error(), &badDoc))

	docID, err := errs[0].DocumentID()
	require.Equal(t, 6, docID)
	require.Nil(t, err)

	require.Empty(t, resFinder.workloads)
	require.Empty(t, resFinder.services)
	require.Empty(t, resFinder.configmaps)
}

func TestGetRelevantK8sResourcesNoK8sResource(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "not_a_k8s_resource.yaml")
	resFinder := newResourceFinder(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	notK8sRes := &NotK8sResourceError{}
	require.True(t, errors.As(errs[0].Error(), &notK8sRes))
	require.Empty(t, resFinder.workloads)
	require.Len(t, resFinder.services, 1)
	require.Empty(t, resFinder.configmaps)
}

func TestGetRelevantK8sResourcesNoYAMLs(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir2")
	resFinder := newResourceFinder(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	noYamls := &NoYamlsFoundError{}
	require.True(t, errors.As(errs[0].Error(), &noYamls))
	require.Empty(t, resFinder.workloads)
	require.Empty(t, resFinder.services)
	require.Empty(t, resFinder.configmaps)
}

func TestGetRelevantK8sResourcesBadDir(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	resFinder := newResourceFinder(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDir := &FailedAccessingDirError{}
	require.True(t, errors.As(errs[0].Error(), &badDir))
	require.Empty(t, resFinder.workloads)
	require.Empty(t, resFinder.services)
	require.Empty(t, resFinder.configmaps)
}

func TestGetRelevantK8sResourcesBadDirFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	resFinder := newResourceFinder(NewDefaultLogger(), true, filepath.WalkDir)
	errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDir := &FailedAccessingDirError{}
	require.True(t, errors.As(errs[0].Error(), &badDir))
	require.Empty(t, resFinder.workloads)
	require.Empty(t, resFinder.services)
	require.Empty(t, resFinder.configmaps)
}

func TestGetRelevantK8sResourcesNonK8sResources(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bookinfo")
	resFinder := newResourceFinder(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resFinder.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 2) // Has Istio resources ClusterIssuer and Certificate
	badResource := &NotK8sResourceError{}
	require.True(t, errors.As(errs[0].Error(), &badResource))
}

func TestSearchForManifests(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	resFinder := newResourceFinder(NewDefaultLogger(), false, filepath.WalkDir)
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
	resFinder := newResourceFinder(NewDefaultLogger(), false, nonRecursiveWalk)
	yamlFiles, errs := resFinder.searchForManifests(dirPath)
	require.Empty(t, errs)
	require.Len(t, yamlFiles, 4)
}
