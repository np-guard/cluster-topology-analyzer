/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import "path/filepath"

func stopProcessing(stopOn1stErr bool, errs []FileProcessingError) bool {
	for idx := range errs {
		if errs[idx].IsFatal() || stopOn1stErr && errs[idx].IsSevere() {
			return true
		}
	}

	return false
}

func appendAndLogNewError(errs []FileProcessingError, newErr *FileProcessingError, logger Logger) []FileProcessingError {
	logError(logger, newErr)
	errs = append(errs, *newErr)
	return errs
}

// returns a file path without its prefix base dir
func pathWithoutBaseDir(path, baseDir string) string {
	if path == baseDir { // baseDir is actually a file...
		return filepath.Base(path) // return just the file name
	}

	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return path
	}
	return relPath
}
