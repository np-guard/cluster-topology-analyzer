/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package controller

import (
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/resource"
)

func infosFromPaths(paths []string, recursive bool) ([]*resource.Info, error) {
	fileOption := resource.FilenameOptions{Filenames: paths, Recursive: recursive}
	builder := resource.NewLocalBuilder()
	resourceResult := builder.
		Unstructured().
		ContinueOnError().
		FilenameParam(false, &fileOption).
		Flatten().
		Do()

	return resourceResult.Infos()
}

func infosFromFilePath(path string, logger Logger) ([]*resource.Info, []FileProcessingError) {
	infos, err := infosFromPaths([]string{path}, false)

	fperrors := []FileProcessingError{}
	if err != nil { // it is possible some files could not be parsed - we'll just report them and continue
		if agg, ok := err.(utilerrors.Aggregate); ok {
			for _, scanErr := range agg.Errors() {
				fperrors = appendAndLogNewError(fperrors, failedReadingFile(path, scanErr), logger)
			}
		} else {
			fperrors = appendAndLogNewError(fperrors, failedReadingFile(path, err), logger)
		}
	}

	return infos, fperrors
}
