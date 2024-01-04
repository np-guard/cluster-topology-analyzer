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

func TestParseK8sYamlBadYamlDocument(t *testing.T) {
	badYamlPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	resAcc := newResourceAccumulator(NewDefaultLogger(), false)
	errs := resAcc.parseK8sYaml(badYamlPath)
	require.Len(t, errs, 1)
	badFile := &FailedReadingFileError{}
	require.True(t, errors.As(errs[0].Error(), &badFile))

	require.Len(t, resAcc.workloads, 3)
	require.Len(t, resAcc.services, 3)
	require.Empty(t, resAcc.configmaps)
}

func TestParseK8sYamlBadYamlDocumentFailFast(t *testing.T) {
	badYamlPath := filepath.Join(getTestsDir(), "bad_yamls", "document_with_syntax_error.yaml")
	resAcc := newResourceAccumulator(NewDefaultLogger(), true)
	errs := resAcc.parseK8sYaml(badYamlPath)
	require.Len(t, errs, 1)
	badFile := &FailedReadingFileError{}
	require.True(t, errors.As(errs[0].Error(), &badFile))

	require.Empty(t, resAcc.workloads)
	require.Empty(t, resAcc.services)
	require.Empty(t, resAcc.configmaps)
}

func TestParseK8sYamlNoK8sResource(t *testing.T) {
	yamlPath := filepath.Join(getTestsDir(), "bad_yamls", "not_a_k8s_resource.yaml")
	resAcc := newResourceAccumulator(NewDefaultLogger(), false)
	errs := resAcc.parseK8sYaml(yamlPath)
	require.Len(t, errs, 1)
	fileErr := &FailedReadingFileError{}
	require.True(t, errors.As(errs[0].Error(), &fileErr))
	require.Empty(t, resAcc.workloads)
	require.Len(t, resAcc.services, 1)
	require.Empty(t, resAcc.configmaps)
}

func TestParseK8sYamlNotYAML(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "..", ".gitignore")
	resAcc := newResourceAccumulator(NewDefaultLogger(), false)
	errs := resAcc.parseK8sYaml(dirPath)
	require.Len(t, errs, 1)
	noYamls := &FailedReadingFileError{}
	require.True(t, errors.As(errs[0].Error(), &noYamls))
	require.Empty(t, resAcc.workloads)
	require.Empty(t, resAcc.services)
	require.Empty(t, resAcc.configmaps)
}

func TestParseK8sYamlNoSuchFile(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "no_such_file") // doesn't exist
	resAcc := newResourceAccumulator(NewDefaultLogger(), false)
	errs := resAcc.parseK8sYaml(dirPath)
	require.Len(t, errs, 1)
	badDir := &FailedReadingFileError{}
	require.True(t, errors.As(errs[0].Error(), &badDir))
	require.Empty(t, resAcc.workloads)
	require.Empty(t, resAcc.services)
	require.Empty(t, resAcc.configmaps)
}

func TestParseK8sYamlNonK8sResources(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bookinfo", "bookinfo-certificate.yaml")
	resAcc := newResourceAccumulator(NewDefaultLogger(), false)
	errs := resAcc.parseK8sYaml(dirPath)
	require.Empty(t, errs) // Irrelevant resources such as Certificate are only reported to log - not returned as errors
}
