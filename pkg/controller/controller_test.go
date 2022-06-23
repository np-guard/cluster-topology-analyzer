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

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

// TestOutput calls controller.Start() with an example repo dir tests/onlineboutique/ ,
// checking for the json output to match expected output at tests/expected_output.json
func TestConnectionsOutput(t *testing.T) {
	currentDir, _ := os.Getwd()
	dirPath := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "kubernetes-manifests.yaml")
	outFile := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "output.json")
	expectedOutput := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "expected_output.json")
	args := getTestArgs(dirPath, outFile, false)

	Start(args)

	res, err := compareFiles(expectedOutput, outFile)

	if err != nil {
		t.Fatalf("expected err to be nil, but got %v", err)
	}
	if !res {
		t.Fatalf("expected res to be true, but got false")
	}

	os.Remove(outFile)
}

func TestDirScan(t *testing.T) {
	currentDir, _ := os.Getwd()
	dirPath := filepath.Join(currentDir, "../../", "tests", "onlineboutique")
	outFile := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "output.json")
	expectedOutput := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "expected_dirscan_output.json")
	args := getTestArgs(dirPath, outFile, false)

	Start(args)

	res, err := compareFiles(expectedOutput, outFile)

	if err != nil {
		t.Fatalf("expected err to be nil, but got %v", err)
	}
	if !res {
		t.Fatalf("expected res to be true, but got false")
	}

	os.Remove(outFile)
}

func TestNetpolsJsonOutput(t *testing.T) {
	currentDir, _ := os.Getwd()
	dirPath := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "kubernetes-manifests.yaml")
	outFile := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "output.json")
	expectedOutput := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "expected_netpol_output.json")
	args := getTestArgs(dirPath, outFile, true)

	Start(args)

	res, err := compareFiles(expectedOutput, outFile)

	if err != nil {
		t.Fatalf("expected err to be nil, but got %v", err)
	}
	if !res {
		t.Fatalf("expected res to be true, but got false")
	}

	os.Remove(outFile)
}

func TestNetpolsInterface(t *testing.T) {
	currentDir, _ := os.Getwd()
	dirPath := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "kubernetes-manifests.yaml")
	outFile := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "output.json")
	expectedOutput := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "expected_netpol_output.json")

	netpols, err := PoliciesFromFolderPath(dirPath)
	if err != nil {
		t.Fatalf("expected err to be nil, but got %v", err)
	}
	if len(netpols) == 0 {
		t.Fatalf("expected policies to be non-empty, but got empty")
	}

	buf, _ := json.MarshalIndent(netpols, "", "    ")
	fp, err := os.Create(outFile)
	if err != nil {
		t.Fatalf("failed opening output file: %v", err)
	}
	fp.Write(buf)
	fp.Close()
	res, err := compareFiles(expectedOutput, outFile)
	if err != nil {
		t.Fatalf("expected err to be nil, but got %v", err)
	}
	if !res {
		t.Fatalf("expected res to be true, but got false")
	}

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

func getTestArgs(dirPath, outFile string, netpols bool) common.InArgs {
	args := common.InArgs{}
	emptyStr := ""
	args.DirPath = &dirPath
	args.CommitID = &emptyStr
	args.GitBranch = &emptyStr
	args.GitURL = &emptyStr
	args.OutputFile = &outFile
	args.SynthNetpols = &netpols
	return args
}

func compareFiles(expectedFile, actualFile string) (bool, error) {
	expected_lines, err1 := readLines(expectedFile)
	actual_lines, err2 := readLines(actualFile)
	if err1 != nil || err2 != nil {
		return false, errors.New("error reading lines from file")
	}
	if len(expected_lines) != len(actual_lines) {
		fmt.Printf("Files line count is different: expected: %d, actual: %d", len(expected_lines), len(actual_lines))
		return false, nil
	}

	for i := 0; i < len(expected_lines); i++ {
		line_expected := expected_lines[i]
		line_actual := actual_lines[i]
		if line_expected != line_actual && strings.Index(line_expected, "\"filepath\"") == -1 {
			fmt.Printf("Gap in line %d: expected: %s, actual: %s", i, line_expected, line_actual)
			return false, nil
		}
	}
	return true, nil
}
