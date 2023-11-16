/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package controller

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
)

func parseResourceFromInfo[T interface{}](info *resource.Info) *T {
	obj, ok := info.Object.(*unstructured.Unstructured)
	if !ok {
		return nil
	}

	var rc T
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &rc)
	if err != nil {
		return nil
	}
	return &rc
}
