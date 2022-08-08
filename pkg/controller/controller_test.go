package controller

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOutput calls controller.Start() with an example repo dir tests/onlineboutique/ ,
// checking for the json output to match expected output at tests/expected_output.json
func TestConnectionsOutput(t *testing.T) {
	currentDir, _ := os.Getwd()
	dirPath := filepath.Join(currentDir, "..", "..", "tests", "onlineboutique", "kubernetes-manifests.yaml")
	outFile := filepath.Join(currentDir, "..", "..", "tests", "onlineboutique", "output.json")
	expectedOutput := filepath.Join(currentDir, "..", "..", "tests", "onlineboutique", "expected_output.json")
	args := getTestArgs(dirPath, outFile, false)

	err := Start(args, SilentIgnore)
	assert.NoError(t, err)

	assert.FileExists(t, outFile)
	assert.NoError(t, compareFiles(expectedOutput, outFile))
	// compare file contents
	f1, _ := ioutil.ReadFile(expectedOutput)
	f2, _ := ioutil.ReadFile(outFile)
	assert.Equal(t, string(f1), string(f2))

	os.Remove(outFile)
}

func TestStartDetailedNetpolOutput(t *testing.T) {
	currentDir, _ := os.Getwd()
	filePath := filepath.Join(currentDir, "..", "..", "tests", "onlineboutique", "kubernetes-manifests.yaml")
	outFile := filepath.Join(currentDir, "..", "..", "tests", "onlineboutique", "output.json")
	expectedOutput := filepath.Join(currentDir, "..", "..", "tests", "onlineboutique", "expected_netpol_output.json")

	k8sDeploymentFiles := getK8sDeploymentResources([]string{filePath}, Strict)
	assert.Equal(t, 1, len(k8sDeploymentFiles))
	totalK8sObjects := 0
	for _, o := range k8sDeploymentFiles {
		assert.NoErrorf(t, o.fileReadingError, "file reading error: '%s'", o.ManifestFilepath)
		assert.NoErrorf(t, o.yamlParseError, "yaml parse error: '%s'", o.ManifestFilepath)

		for _, deploy := range o.DeployObjects {
			totalK8sObjects++
			assert.NoErrorf(t, deploy.yamlDocDecodeError, "yaml decoding error for file: '%s'", o.ManifestFilepath)
		}
	}
	assert.Equal(t, 25, totalK8sObjects, "yaml file should contain 25 relevant k8s objects")

	conns, err := extractConnections(k8sDeploymentFiles, "", "", "")
	assert.NoErrorf(t, err, "extracting connections failed")
	err = writeOut(conns, outFile, true)
	assert.NoErrorf(t, err, "writing to output file")
	assert.NoError(t, compareFiles(expectedOutput, outFile))
	// compare file contents
	f1, _ := ioutil.ReadFile(expectedOutput)
	f2, _ := ioutil.ReadFile(outFile)
	assert.Equal(t, string(f1), string(f2))
	os.Remove(outFile)
}

func TestDirScan(t *testing.T) {
	currentDir, _ := os.Getwd()
	dirPath := filepath.Join(currentDir, "..", "..", "tests", "onlineboutique")
	outFile := filepath.Join(currentDir, "..", "..", "tests", "onlineboutique", "output.json")
	expectedOutput := filepath.Join(currentDir, "..", "..", "tests", "onlineboutique", "expected_dirscan_output.json")
	args := getTestArgs(dirPath, outFile, false)

	err := Start(args, SilentIgnore)
	assert.NoError(t, err)
	assert.NoError(t, compareFiles(expectedOutput, outFile))
	os.Remove(outFile)
}

type TestDetails struct {
	dirPath        string
	outFile        string
	expectedOutput string
}

func TestNetpolsJsonOutput(t *testing.T) {
	currentDir, _ := os.Getwd()
	testsDir := filepath.Join(currentDir, "..", "..", "tests")
	tests := map[string]TestDetails{} // map from test name to test details
	tests["onlineboutique"] = TestDetails{dirPath: filepath.Join(testsDir, "onlineboutique", "kubernetes-manifests.yaml"),
		outFile:        filepath.Join(testsDir, "onlineboutique", "output.json"),
		expectedOutput: filepath.Join(testsDir, "onlineboutique", "expected_netpol_output.json")}
	tests["sockshop"] = TestDetails{dirPath: filepath.Join(testsDir, "sockshop", "manifests"),
		outFile:        filepath.Join(testsDir, "sockshop", "output.json"),
		expectedOutput: filepath.Join(testsDir, "sockshop", "expected_netpol_output.json")}
	tests["wordpress"] = TestDetails{dirPath: filepath.Join(testsDir, "k8s_wordpress_example"),
		outFile:        filepath.Join(testsDir, "k8s_wordpress_example", "output.json"),
		expectedOutput: filepath.Join(testsDir, "k8s_wordpress_example", "expected_netpol_output.json")}
	tests["guestbook"] = TestDetails{dirPath: filepath.Join(testsDir, "k8s_guestbook"),
		outFile:        filepath.Join(testsDir, "k8s_guestbook", "output.json"),
		expectedOutput: filepath.Join(testsDir, "k8s_guestbook", "expected_netpol_output.json")}

	for testName, testDetails := range tests {
		t.Run(testName, func(t *testing.T) {
			args := getTestArgs(testDetails.dirPath, testDetails.outFile, true)
			err := Start(args, SilentIgnore)
			assert.NoError(t, err)
			assert.NoError(t, compareFiles(testDetails.expectedOutput, testDetails.outFile))

			os.Remove(testDetails.outFile)
		})
	}
}

func TestNetpolsInterface(t *testing.T) {
	currentDir, _ := os.Getwd()
	testsDir := filepath.Join(currentDir, "..", "..", "tests")
	dirPath := filepath.Join(testsDir, "onlineboutique", "kubernetes-manifests.yaml")
	outFile := filepath.Join(testsDir, "onlineboutique", "output.json")
	expectedOutput := filepath.Join(testsDir, "onlineboutique", "expected_netpol_interface_output.json")

	netpols, err := PoliciesFromFolderPath(dirPath, SilentIgnore)
	if err != nil {
		t.Fatalf("expected fileReadingError to be nil, but got %v", err)
	}
	if len(netpols) == 0 {
		t.Fatalf("expected policies to be non-empty, but got empty")
	}

	buf, _ := json.MarshalIndent(netpols, "", "    ")
	fp, err := os.Create(outFile)
	if err != nil {
		t.Fatalf("failed opening output file: %v", err)
	}
	_, err = fp.Write(buf)
	if err != nil {
		t.Fatalf("failed writing to output file: %v", err)
	}
	fp.Close()
	assert.NoError(t, compareFiles(expectedOutput, outFile))

	os.Remove(outFile)
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

func getTestArgs(dirPath, outFile string, netpols bool) InArgs {
	emptyStr := ""
	return InArgs{
		DirPath:      &dirPath,
		CommitID:     &emptyStr,
		GitURL:       &emptyStr,
		OutputFile:   &outFile,
		SynthNetpols: &netpols,
	}
}

func compareFiles(expectedFile, actualFile string) error {
	expectedLines, err1 := readLines(expectedFile)
	actualLines, err2 := readLines(actualFile)
	if err1 != nil || err2 != nil {
		return errors.New("error reading lines from either files")
	}
	if len(expectedLines) != len(actualLines) {
		return fmt.Errorf("Files line count is different: expected(%s): %d, actual(%s): %d",
			expectedFile, len(expectedLines), actualFile, len(actualLines))
	}

	for i := 0; i < len(expectedLines); i++ {
		lineExpected := expectedLines[i]
		lineActual := actualLines[i]
		if lineExpected != lineActual && !strings.Contains(lineExpected, "\"filepath\"") {
			return fmt.Errorf("Gap in line %d: expected(%s): %s, actual(%s): %s", i, expectedFile, lineExpected, actualFile, lineActual)
		}
	}
	return nil
}
