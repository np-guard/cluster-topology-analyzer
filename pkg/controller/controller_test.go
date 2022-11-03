package controller

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
)

// TestOutput calls controller.Start() with an example repo dir tests/onlineboutique/ ,
// checking for the json output to match expected output at tests/expected_output.json

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

func TestPoliciesSynthesizerAPIFatalError(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "badPath")
	logger := NewDefaultLogger()
	synthesizer := NewPoliciesSynthesizer(WithLogger(logger))
	netpols, err := synthesizer.PoliciesFromFolderPath(dirPath)
	require.NotNil(t, err)
	require.Len(t, synthesizer.Errors(), 1)
	require.Empty(t, netpols)
}

func TestPoliciesSynthesizerAPIFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	synthesizer := NewPoliciesSynthesizer(WithStopOnError())
	netpols, err := synthesizer.PoliciesFromFolderPath(dirPath)
	require.Nil(t, err)
	require.Len(t, synthesizer.Errors(), 1)
	require.Empty(t, netpols)
}

func TestExtractConnectionsNoK8sResources(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "irrelevant_k8s_resources.yaml")
	resources, conns, errs := extractConnections(dirPath, false)
	require.Len(t, errs, 1)
	require.Empty(t, conns)
	require.Empty(t, resources)
}

func TestExtractConnectionsNoK8sResourcesFailFast(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls")
	resources, conns, errs := extractConnections(dirPath, true)
	require.Len(t, errs, 1)
	require.Empty(t, conns)
	require.Empty(t, resources)
}

func TestExtractConnectionsBadConfigMapRefs(t *testing.T) {
	dirPath := filepath.Join(getTestsDir(), "bad_yamls", "bad_configmap_refs.yaml")
	resources, conns, errs := extractConnections(dirPath, false)
	require.Len(t, errs, 3)
	require.Empty(t, conns)
	require.Len(t, resources, 2) // the two deployments in this example get read
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
