/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRelevantK8sResourcesBadYamlDocument(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	resAcc := newResourceAccumulator(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resAcc.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badFile := &FailedReadingFileError{}
	require.True(t, errors.As(errs[0].Error(), &badFile))

	require.Len(t, resAcc.workloads, 3)
	require.Len(t, resAcc.services, 3)
	require.Empty(t, resAcc.configmaps)
}

func TestGetRelevantK8sResourcesBadYamlDocumentFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	resAcc := newResourceAccumulator(NewDefaultLogger(), true, filepath.WalkDir)
	errs := resAcc.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badFile := &FailedReadingFileError{}
	require.True(t, errors.As(errs[0].Error(), &badFile))

	require.Empty(t, resAcc.workloads)
	require.Empty(t, resAcc.services)
	require.Empty(t, resAcc.configmaps)
}

func TestGetRelevantK8sResourcesNoK8sResource(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "not_a_k8s_resource.yaml")
	resAcc := newResourceAccumulator(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resAcc.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	fileErr := &FailedReadingFileError{}
	require.True(t, errors.As(errs[0].Error(), &fileErr))
	require.Empty(t, resAcc.workloads)
	require.Len(t, resAcc.services, 1)
	require.Empty(t, resAcc.configmaps)
}

func TestGetRelevantK8sResourcesNoYAMLs(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir2")
	resAcc := newResourceAccumulator(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resAcc.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	noYamls := &NoYamlsFoundError{}
	require.True(t, errors.As(errs[0].Error(), &noYamls))
	require.Empty(t, resAcc.workloads)
	require.Empty(t, resAcc.services)
	require.Empty(t, resAcc.configmaps)
}

func TestGetRelevantK8sResourcesBadDir(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	resAcc := newResourceAccumulator(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resAcc.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDir := &FailedAccessingDirError{}
	require.True(t, errors.As(errs[0].Error(), &badDir))
	require.Empty(t, resAcc.workloads)
	require.Empty(t, resAcc.services)
	require.Empty(t, resAcc.configmaps)
}

func TestGetRelevantK8sResourcesBadDirFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir3") // doesn't exist
	resAcc := newResourceAccumulator(NewDefaultLogger(), true, filepath.WalkDir)
	errs := resAcc.getRelevantK8sResources(dirPath)
	require.Len(t, errs, 1)
	badDir := &FailedAccessingDirError{}
	require.True(t, errors.As(errs[0].Error(), &badDir))
	require.Empty(t, resAcc.workloads)
	require.Empty(t, resAcc.services)
	require.Empty(t, resAcc.configmaps)
}

func TestGetRelevantK8sResourcesNonK8sResources(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bookinfo")
	resAcc := newResourceAccumulator(NewDefaultLogger(), false, filepath.WalkDir)
	errs := resAcc.getRelevantK8sResources(dirPath)
	require.Empty(t, errs) // Irrelevant resources such as Certificate are only reported to log - not returned as errors
}
