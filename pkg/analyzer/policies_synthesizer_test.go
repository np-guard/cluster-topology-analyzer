/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package analyzer

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"

	"github.com/np-guard/netpol-analyzer/pkg/manifests/fsscanner"
)

func TestPoliciesSynthesizerAPI(t *testing.T) {
	testsDir := getTestsDir()
	dirPath := filepath.Join(testsDir, "onlineboutique", "kubernetes-manifests.yaml")
	outFile := filepath.Join(testsDir, "onlineboutique", "output.json")
	expectedOutput := filepath.Join(testsDir, "onlineboutique", "expected_netpol_interface_output.json")

	logger := NewDefaultLogger()
	synthesizer := NewPoliciesSynthesizer(WithLogger(logger))
	netpols, err := synthesizer.PoliciesFromFolderPath(dirPath)
	require.Nilf(t, err, "expected no fatal errors, but got %v", err)

	fileScanningErrors := synthesizer.Errors()
	require.Empty(t, fileScanningErrors)
	require.NotEmpty(t, netpols)

	buf, _ := json.MarshalIndent(netpols, "", "    ")
	fp, err := os.Create(outFile)
	require.Nil(t, err, "failed opening output file")
	_, err = fp.Write(buf)
	require.Nil(t, err, "failed writing to output file")
	fp.Close()

	res, err := compareFiles(expectedOutput, outFile)
	require.Nil(t, err, "error comparing files")
	require.True(t, res, "files not equal")

	os.Remove(outFile)
}

func TestPoliciesSynthesizerAPIWithInfos(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "k8s_wordpress_example")
	infos, errs := fsscanner.GetResourceInfosFromDirPath([]string{dirPath}, true, false)
	require.Empty(t, errs)

	synthesizer := NewPoliciesSynthesizer()
	policies, err := synthesizer.PoliciesFromInfos(infos)
	require.Nil(t, err)
	require.Empty(t, synthesizer.Errors())
	require.Len(t, policies, 3) // wordpress, mysql and namespace default deny

	conns, err := synthesizer.ConnectionsFromInfos(infos)
	require.Nil(t, err)
	require.Empty(t, synthesizer.Errors())
	require.Len(t, conns, 2) // internet->wordpress and wordpress->mysql
}

func TestPoliciesSynthesizerAPIWithInfosEmptySlice(t *testing.T) {
	noInfos := []*resource.Info{}

	synthesizer := NewPoliciesSynthesizer()
	_, err := synthesizer.PoliciesFromInfos(noInfos)
	require.NotNil(t, err)

	_, err = synthesizer.ConnectionsFromInfos(noInfos)
	require.NotNil(t, err)
}

func TestPoliciesSynthesizerAPIWithInfosBadInfo(t *testing.T) {
	badInfo1 := resource.Info{}
	badInfo2 := resource.Info{Object: &unstructured.Unstructured{}}
	badInfo3 := resource.Info{Object: &unstructured.Unstructured{Object: map[string]interface{}{"kind": "bad"}}}
	badInfo4 := resource.Info{Object: &unstructured.Unstructured{Object: map[string]interface{}{"kind": "Service", "spec": []string{}}}}
	badInfos := []*resource.Info{&badInfo1, &badInfo2, &badInfo3, &badInfo4}

	synthesizer := NewPoliciesSynthesizer()
	_, err := synthesizer.PoliciesFromInfos(badInfos)
	require.NotNil(t, err)

	_, err = synthesizer.ConnectionsFromInfos(badInfos)
	require.NotNil(t, err)
}

func TestPoliciesSynthesizerAPIMultiplePaths(t *testing.T) {
	dirPath1 := filepath.Join(getTestsDir(), "k8s_wordpress_example", "mysql-deployment.yaml")
	dirPath2 := filepath.Join(getTestsDir(), "k8s_wordpress_example", "wordpress-deployment.yaml")
	synthesizer := NewPoliciesSynthesizer()
	netpols, err := synthesizer.PoliciesFromFolderPaths([]string{dirPath1, dirPath2})
	require.Nilf(t, err, "expected no fatal errors, but got %v", err)
	require.Empty(t, synthesizer.Errors())
	require.Len(t, netpols, 3)

	conns, err := synthesizer.ConnectionsFromFolderPath(dirPath2)
	require.Nilf(t, err, "expected no fatal errors, but got %v", err)
	require.Empty(t, synthesizer.Errors())
	require.Len(t, conns, 1)
}

func TestPoliciesSynthesizerAPIDnsPort(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "acs-security-demos")
	synthesizer := NewPoliciesSynthesizer(WithDNSPort(5353))
	netpols, err := synthesizer.PoliciesFromFolderPaths([]string{dirPath})
	require.Nilf(t, err, "expected no fatal errors, but got %v", err)
	require.Empty(t, synthesizer.Errors())
	require.Len(t, netpols, 14)
	for _, netpol := range netpols {
		for r := range netpol.Spec.Egress {
			eRule := &netpol.Spec.Egress[r]
			for p := range eRule.Ports {
				port := &eRule.Ports[p]
				if *port.Protocol == core.ProtocolUDP {
					require.Equal(t, int32(5353), port.Port.IntVal)
				}
			}
		}
	}
}

func TestPoliciesSynthesizerConnectionsFile(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "sockshop", "manifests")
	connFilePath := filepath.Join(getTestsDir(), "sockshop", "connections.txt")
	synthesizer := NewPoliciesSynthesizer(WithConnectionsFile(connFilePath))
	netpols, err := synthesizer.PoliciesFromFolderPaths([]string{dirPath})
	require.Nil(t, err)
	require.Len(t, netpols, 15)
}

func TestPoliciesSynthesizerAPIFatalError(t *testing.T) {
	dirPath1 := filepath.Join(getTestsDir(), "k8s_wordpress_example")
	dirPath2 := filepath.Join(getTestsDir(), "badPath")
	logger := NewDefaultLogger()
	synthesizer := NewPoliciesSynthesizer(WithLogger(logger))
	netpols, err := synthesizer.PoliciesFromFolderPaths([]string{dirPath1, dirPath2})
	badDir := &FailedAccessingDirError{}
	require.NotNil(t, err)
	require.True(t, errors.As(err, &badDir))
	require.Len(t, synthesizer.Errors(), 1)
	require.True(t, errors.As(synthesizer.Errors()[0].Error(), &badDir))
	require.Empty(t, netpols)
}

func TestPoliciesSynthesizerAPIFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	synthesizer := NewPoliciesSynthesizer(WithStopOnError())
	netpols, err := synthesizer.PoliciesFromFolderPath(dirPath)
	require.Nil(t, err)
	require.Len(t, synthesizer.Errors(), 1)
	badYaml := &FailedReadingFileError{}
	require.True(t, errors.As(synthesizer.Errors()[0].Error(), &badYaml))
	require.Len(t, synthesizer.ErrorPtrs(), 1)
	require.True(t, errors.As(synthesizer.ErrorPtrs()[0].Error(), &badYaml))
	require.Empty(t, netpols)
}

func TestExtractConnectionsNoK8sResources(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "irrelevant_k8s_resources.yaml")
	synthesizer := NewPoliciesSynthesizer()
	resources, conns, errs := synthesizer.extractConnectionsFromFolderPaths([]string{dirPath})
	require.Len(t, errs, 1)
	noK8sRes := &NoK8sResourcesFoundError{}
	require.True(t, errors.As(errs[0].Error(), &noK8sRes))
	require.Empty(t, conns)
	require.Empty(t, resources)
}

func TestExtractConnectionsNoK8sResourcesFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	synthesizer := NewPoliciesSynthesizer(WithStopOnError())
	resources, conns, errs := synthesizer.extractConnectionsFromFolderPaths([]string{dirPath})
	require.Len(t, errs, 1)
	require.Empty(t, conns)
	require.Empty(t, resources)
}

func TestExtractConnectionsBadConfigMapRefs(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "bad_configmap_refs.yaml")
	synthesizer := NewPoliciesSynthesizer()
	resources, conns, errs := synthesizer.extractConnectionsFromFolderPaths([]string{dirPath})
	require.Len(t, errs, 3)
	noConfigMap := &ConfigMapNotFoundError{}
	noConfigMapKey := &ConfigMapKeyNotFoundError{}
	for _, err := range errs {
		require.True(t, errors.As(err.Error(), &noConfigMap) || errors.As(err.Error(), &noConfigMapKey))
	}
	require.Empty(t, conns)
	require.Len(t, resources, 2) // the two deployments in this example get read
}

func TestExtractConnectionsCustomWalk(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "sockshop")
	synthesizer := NewPoliciesSynthesizer(WithWalkFn(nonRecursiveWalk))
	resources, conns, errs := synthesizer.extractConnectionsFromFolderPaths([]string{dirPath})
	require.Len(t, errs, 2) // no yaml should be found in a non-recursive scan
	noYamls := &NoYamlsFoundError{}
	noK8sRes := &NoK8sResourcesFoundError{}
	require.True(t, errors.As(errs[0].Error(), &noYamls))
	require.True(t, errors.As(errs[1].Error(), &noK8sRes))
	require.Empty(t, conns)
	require.Empty(t, resources)
}

func TestExtractConnectionsCustomWalk2(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "sockshop")
	synthesizer := NewPoliciesSynthesizer(WithWalkFn(filepath.WalkDir))
	resources, conns, errs := synthesizer.extractConnectionsFromFolderPaths([]string{dirPath})
	require.Len(t, errs, 0)
	require.Len(t, conns, 14)
	require.Len(t, resources, 14)
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func getTestsDir() string {
	currentDir, _ := os.Getwd()
	return filepath.Join(currentDir, "..", "..", "tests")
}

func compareFiles(expectedFile, actualFile string) (bool, error) {
	expectedLines, err1 := readLines(expectedFile)
	actualLines, err2 := readLines(actualFile)
	if err1 != nil || err2 != nil {
		return false, errors.New("error reading lines from file")
	}
	if len(expectedLines) != len(actualLines) {
		fmt.Printf("Files line count is different: expected(%s): %d, actual(%s): %d",
			expectedFile, len(expectedLines), actualFile, len(actualLines))
		return false, nil
	}

	for i := 0; i < len(expectedLines); i++ {
		lineExpected := expectedLines[i]
		lineActual := actualLines[i]
		if lineExpected != lineActual && !strings.Contains(lineExpected, "\"filepath\"") {
			fmt.Printf("Gap in line %d: expected(%s): %s, actual(%s): %s", i, expectedFile, lineExpected, actualFile, lineActual)
			return false, nil
		}
	}
	return true, nil
}
