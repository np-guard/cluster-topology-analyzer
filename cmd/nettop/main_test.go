/*
Copyright 2020- IBM Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestDetails struct {
	name           string
	dirPath        [][]string
	outputFormat   string
	synthNetpols   bool
	extraFlags     []string
	expectError    bool
	expectedOutput []string
}

var (
	testCaseScenarios = []TestDetails{
		{
			"ConnectionsOutputJSON",
			[][]string{{"onlineboutique", "kubernetes-manifests.yaml"}},
			jsonFormat,
			false,
			nil,
			false,
			[]string{"onlineboutique", "expected_output.json"},
		},
		{
			"ConnectionsOutputYAML",
			[][]string{{"onlineboutique", "kubernetes-manifests.yaml"}},
			yamlFormat,
			false,
			nil,
			false,
			[]string{"onlineboutique", "expected_output.yaml"},
		},
		{
			"DirScan",
			[][]string{{"onlineboutique"}},
			jsonFormat,
			false,
			[]string{"-q"},
			false,
			[]string{"onlineboutique", "expected_dirscan_output.json"},
		},
		{
			"NetpolsOnlineBoutiqueYAML",
			[][]string{{"onlineboutique", "kubernetes-manifests.yaml"}},
			yamlFormat,
			true,
			nil,
			false,
			[]string{"onlineboutique", "expected_netpol_output.yaml"},
		},
		{
			"NetpolsFromPodsOnlineBoutiqueYAML",
			[][]string{{"onlineboutique-pods"}},
			yamlFormat,
			true,
			nil,
			false,
			[]string{"onlineboutique-pods", "expected_netpol_output.yaml"},
		},
		{
			"NetpolsMultiplePaths",
			[][]string{{"k8s_wordpress_example", "mysql-deployment.yaml"}, {"k8s_wordpress_example", "wordpress-deployment.yaml"}},
			jsonFormat,
			true,
			nil,
			false,
			[]string{"k8s_wordpress_example", "expected_netpol_output.json"},
		},
		{
			"NetpolsOnlineBoutiqueJson",
			[][]string{{"onlineboutique", "kubernetes-manifests.yaml"}},
			jsonFormat,
			true,
			nil,
			false,
			[]string{"onlineboutique", "expected_netpol_output.json"},
		},
		{
			"NetpolsSockshop",
			[][]string{{"sockshop", "manifests"}},
			jsonFormat,
			true,
			[]string{"-v"},
			false,
			[]string{"sockshop", "expected_netpol_output.json"},
		},
		{
			"NetpolsK8sWordpress",
			[][]string{{"k8s_wordpress_example"}},
			jsonFormat,
			true,
			[]string{"-v"},
			false,
			[]string{"k8s_wordpress_example", "expected_netpol_output.json"},
		},
		{
			"NetpolsK8sGuestbook",
			[][]string{{"k8s_guestbook"}},
			jsonFormat,
			true,
			[]string{"-v"},
			false,
			[]string{"k8s_guestbook", "expected_netpol_output.json"},
		},
		{
			"NetpolsBookInfo",
			[][]string{{"bookinfo"}},
			jsonFormat,
			true,
			[]string{"-v"},
			false,
			[]string{"bookinfo", "expected_netpol_output.json"},
		},
		{
			"QuoteOfTheDay",
			[][]string{{"qotd"}},
			jsonFormat,
			true,
			[]string{"-v"},
			false,
			[]string{"qotd", "expected_netpol_output.json"},
		},
		{
			"ScoreDemo-with-gateway-api",
			[][]string{{"score-demo"}},
			yamlFormat,
			true,
			[]string{"-v"},
			false,
			[]string{"score-demo", "expected_netpol_output.yaml"},
		},
		{
			"CronJobWithNontrivialNetworkAddresses",
			[][]string{{"openshift", "openshift-operator-lifecycle-manager-resources.yaml"}},
			yamlFormat,
			true,
			[]string{"-v"},
			false,
			[]string{"openshift", "expected_netpol_output.yaml"},
		},
		{
			"SpecifyDNSPort",
			[][]string{{"acs-security-demos"}},
			yamlFormat,
			true,
			[]string{"-v", "-dnsport", "5353"},
			false,
			[]string{"acs-security-demos", "expected_netpol_output.yaml"},
		},
		{
			"HelpFlag",
			nil,
			jsonFormat,
			true,
			[]string{"-h"},
			false,
			nil,
		},
		{
			"BadFlag",
			[][]string{{"bookinfo"}},
			jsonFormat,
			true,
			[]string{"-no_such_flag"},
			true,
			nil,
		},
		{
			"QuietAndVerbose",
			[][]string{{"bookinfo"}},
			jsonFormat,
			true,
			[]string{"-q", "-v"},
			true,
			nil,
		},
		{
			"BadOutputFormat",
			[][]string{{"bookinfo"}},
			"StrangeFormat",
			true,
			nil,
			true,
			nil,
		},
		{
			"noDirPath",
			nil,
			jsonFormat,
			true,
			nil,
			true,
			nil,
		},
		{
			"badDirPathConnections",
			[][]string{{"no-such-path"}},
			jsonFormat,
			false,
			nil,
			true,
			nil,
		},
		{
			"badDirPathNetpols",
			[][]string{{"no-such-path"}},
			jsonFormat,
			true,
			nil,
			true,
			nil,
		},
		{
			"badYamls",
			[][]string{{"bad_yamls"}},
			jsonFormat,
			true,
			[]string{"-v"},
			false,
			nil,
		},
	}

	currentDir, _ = os.Getwd()
	testsDir      = filepath.Join(currentDir, "..", "..", "tests")
)

func (td *TestDetails) runTest(t *testing.T) {
	t.Logf("Running test %s", td.name)
	outFileName, err := getTempOutputFile()
	require.Nil(t, err)

	testArgs := getTestArgs(td, outFileName)
	t.Logf("Test args: %v", testArgs)
	err = _main(testArgs)

	if td.expectError {
		require.NotNil(t, err)
	} else {
		require.Nil(t, err)
		if td.expectedOutput != nil {
			res, err := compareFiles(pathInTestsDir(td.expectedOutput), outFileName)
			require.Nil(t, err)
			require.True(t, res)

			os.Remove(outFileName)
		}
	}
}

func TestAll(t *testing.T) {
	for testIdx := range testCaseScenarios {
		tc := &testCaseScenarios[testIdx] // rebind tc into this lexical scope to support reentrancy
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.runTest(t)
		})
	}
}

func getTempOutputFile() (string, error) {
	outFile, err := os.CreateTemp(os.TempDir(), "cta_temp")
	if err != nil {
		return "", err
	}
	outFileName := outFile.Name()
	err = outFile.Close()
	return outFileName, err
}

func pathInTestsDir(pathElements []string) string {
	return filepath.Join(testsDir, filepath.Join(pathElements...))
}

func getTestArgs(td *TestDetails, outFile string) []string {
	res := []string{"-outputfile", outFile, "-format", td.outputFormat}
	if td.synthNetpols {
		res = append(res, "-netpols")
	}
	for idx := range td.dirPath {
		res = append(res, "-dirpath", pathInTestsDir(td.dirPath[idx]))
	}
	res = append(res, td.extraFlags...)
	return res
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

func compareFiles(expectedFile, actualFile string) (bool, error) {
	expectedLines, err1 := readLines(expectedFile)
	if err1 != nil {
		return false, fmt.Errorf("error reading lines from file %w", err1)
	}
	actualLines, err2 := readLines(actualFile)
	if err2 != nil {
		return false, fmt.Errorf("error reading lines from file %w", err2)
	}
	if len(expectedLines) != len(actualLines) {
		fmt.Printf("Files line count is different: expected(%s): %d, actual(%s): %d",
			expectedFile, len(expectedLines), actualFile, len(actualLines))
		return false, nil
	}

	for i := 0; i < len(expectedLines); i++ {
		lineExpected := expectedLines[i]
		lineActual := actualLines[i]
		if lineExpected != lineActual && !strings.Contains(lineExpected, "filepath") {
			fmt.Printf("Gap in line %d:\n  expected(%s): %s\n  actual(%s): %s\n", i, expectedFile, lineExpected, actualFile, lineActual)
			return false, nil
		}
	}
	return true, nil
}
