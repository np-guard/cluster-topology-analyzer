package controller

import (
	"k8s.io/apimachinery/pkg/types"
)

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

func namespacedName(namespace, resource string) string {
	namespacedRes := types.NamespacedName{Namespace: namespace, Name: resource}
	return namespacedRes.String()
}
