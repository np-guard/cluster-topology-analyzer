/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSearchForManifests(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	manFinder := manifestFinder{NewDefaultLogger(), false, filepath.WalkDir}
	yamlFiles, errs := manFinder.searchForManifests(dirPath)
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
	yamlFiles, errs := manFinder.searchForManifests(dirPath)
	require.Empty(t, errs)
	require.Len(t, yamlFiles, 4)
}
