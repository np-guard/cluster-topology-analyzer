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
	quiet          bool
	verbose        bool
	expectError    bool
	expectedOutput []string
}

var (
	testCaseScenarios = []TestDetails{
		{
			"ConnectionsOutputJSON",
			[][]string{{"onlineboutique", "kubernetes-manifests.yaml"}},
			JSONFormat,
			false,
			false,
			false,
			false,
			[]string{"onlineboutique", "expected_output.json"},
		},
		{
			"ConnectionsOutputYAML",
			[][]string{{"onlineboutique", "kubernetes-manifests.yaml"}},
			YamlFormat,
			false,
			false,
			false,
			false,
			[]string{"onlineboutique", "expected_output.yaml"},
		},
		{
			"DirScan",
			[][]string{{"onlineboutique"}},
			JSONFormat,
			false,
			true,
			false,
			false,
			[]string{"onlineboutique", "expected_dirscan_output.json"},
		},
		{
			"NetpolsOnlineBoutiqueYAML",
			[][]string{{"onlineboutique", "kubernetes-manifests.yaml"}},
			YamlFormat,
			true,
			false,
			false,
			false,
			[]string{"onlineboutique", "expected_netpol_output.yaml"},
		},
		{
			"NetpolsMultiplePaths",
			[][]string{{"k8s_wordpress_example", "mysql-deployment.yaml"}, {"k8s_wordpress_example", "wordpress-deployment.yaml"}},
			JSONFormat,
			true,
			false,
			false,
			false,
			[]string{"k8s_wordpress_example", "expected_netpol_output.json"},
		},
		{
			"NetpolsOnlineBoutiqueJson",
			[][]string{{"onlineboutique", "kubernetes-manifests.yaml"}},
			JSONFormat,
			true,
			false,
			false,
			false,
			[]string{"onlineboutique", "expected_netpol_output.json"},
		},
		{
			"NetpolsSockshop",
			[][]string{{"sockshop", "manifests"}},
			JSONFormat,
			true,
			false,
			true,
			false,
			[]string{"sockshop", "expected_netpol_output.json"},
		},
		{
			"NetpolsK8sWordpress",
			[][]string{{"k8s_wordpress_example"}},
			JSONFormat,
			true,
			false,
			true,
			false,
			[]string{"k8s_wordpress_example", "expected_netpol_output.json"},
		},
		{
			"NetpolsK8sGuestbook",
			[][]string{{"k8s_guestbook"}},
			JSONFormat,
			true,
			false,
			true,
			false,
			[]string{"k8s_guestbook", "expected_netpol_output.json"},
		},
		{
			"NetpolsBookInfo",
			[][]string{{"bookinfo"}},
			JSONFormat,
			true,
			false,
			true,
			false,
			[]string{"bookinfo", "expected_netpol_output.json"},
		},
	}

	currentDir, _ = os.Getwd()
	testsDir      = filepath.Join(currentDir, "..", "..", "tests")
)

func (td *TestDetails) runTest(t *testing.T) {
	t.Logf("Running test %s", td.name)
	outFileName, err := getTempOutputFile()
	require.Nil(t, err)

	args := getTestArgs(td.dirPath, outFileName, td.outputFormat, td.synthNetpols, td.quiet, td.verbose)
	err = detectTopology(args)

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

func getTestArgs(dirPaths [][]string, outFile, outFormat string, netpols, quiet, verbose bool) InArgs {
	args := InArgs{}
	args.DirPaths = []string{}
	for idx := range dirPaths {
		args.DirPaths = append(args.DirPaths, pathInTestsDir(dirPaths[idx]))
	}
	args.OutputFile = &outFile
	args.OutputFormat = &outFormat
	args.SynthNetpols = &netpols
	args.Quiet = &quiet
	args.Verbose = &verbose
	return args
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
		return false, fmt.Errorf("error reading lines from file %v", err1)
	}
	actualLines, err2 := readLines(actualFile)
	if err2 != nil {
		return false, fmt.Errorf("error reading lines from file %v", err2)
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
