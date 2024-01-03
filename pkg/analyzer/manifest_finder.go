/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"os"
	"path/filepath"
	"regexp"
)

var yamlSuffix = regexp.MustCompile(".ya?ml$")

// manifestFinder is a utility class for searching for YAML files
type manifestFinder struct {
	logger       Logger
	stopOn1stErr bool
	walkFn       WalkFunction // for customizing directory scan
}

// searchForManifests returns a list of YAML files under a given directory.
// Directory is scanned using the configured walk function
func (mf *manifestFinder) searchForManifests(repoDir string) ([]string, []FileProcessingError) {
	yamls := []string{}
	errors := []FileProcessingError{}
	err := mf.walkFn(repoDir, func(path string, f os.DirEntry, err error) error {
		if err != nil {
			errors = appendAndLogNewError(errors, failedAccessingDir(path, err, path != repoDir), mf.logger)
			if stopProcessing(mf.stopOn1stErr, errors) {
				return err
			}
			return filepath.SkipDir
		}
		if f != nil && !f.IsDir() && yamlSuffix.MatchString(f.Name()) {
			yamls = append(yamls, path)
		}
		return nil
	})
	if err != nil {
		mf.logger.Errorf(err, "Error walking directory: %v", err)
	}
	return yamls, errors
}
