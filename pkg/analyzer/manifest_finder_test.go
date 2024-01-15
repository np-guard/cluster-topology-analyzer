/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchForManifests(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	manFinder := manifestFinder{NewDefaultLogger(), false, filepath.WalkDir}
	yamlFiles, errs := manFinder.searchForManifestsInDir(dirPath)
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
	manFinder := manifestFinder{NewDefaultLogger(), false, nonRecursiveWalk}
	yamlFiles, errs := manFinder.searchForManifestsInDir(dirPath)
	require.Empty(t, errs)
	require.Len(t, yamlFiles, 4)
}

func TestSearchForManifestsMultipleDirs(t *testing.T) {
	dirPath1 := filepath.Join(getTestsDir(), "k8s_wordpress_example")
	dirPath2 := filepath.Join(getTestsDir(), "onlineboutique")
	manFinder := manifestFinder{NewDefaultLogger(), false, filepath.WalkDir}
	yamlFiles, errs := manFinder.searchForManifestsInDirs([]string{dirPath1, dirPath2})
	require.Empty(t, errs)
	require.Len(t, yamlFiles, 5)
}

func TestSearchForManifestsMultipleDirsWithErrors(t *testing.T) {
	dirPath1 := filepath.Join(getTestsDir(), "k8s_wordpress_example")
	dirPath2 := filepath.Join(getTestsDir(), "badPath")
	manFinder := manifestFinder{NewDefaultLogger(), false, filepath.WalkDir}
	yamlFiles, errs := manFinder.searchForManifestsInDirs([]string{dirPath1, dirPath2})
	badDir := &FailedAccessingDirError{}
	require.NotEmpty(t, errs)
	require.True(t, errors.As(errs[0].Error(), &badDir))
	require.Empty(t, yamlFiles)
}

func TestNoYamlsInDir(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "subdir2")
	manFinder := manifestFinder{NewDefaultLogger(), false, filepath.WalkDir}
	yamlFiles, errs := manFinder.searchForManifestsInDirs([]string{dirPath})
	require.Len(t, errs, 1)
	noYamls := &NoYamlsFoundError{}
	require.True(t, errors.As(errs[0].Error(), &noYamls))
	require.Empty(t, yamlFiles)
}
