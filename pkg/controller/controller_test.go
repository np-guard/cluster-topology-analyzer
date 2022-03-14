package controller

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.ibm.com/gitsecure-net-top/pkg/common"
)

// TestOutput calls controller.Start() with an example repo dir tests/onlineboutique/ ,
// checking for the json output to match expected output at tests/expected_output.json
func TestOutput(t *testing.T) {
	currentDir, _ := os.Getwd()
	dirPath := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "kubernetes-manifests.yaml")
	outFile := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "output.json")
	expectedOutput := filepath.Join(currentDir, "../../", "tests", "onlineboutique", "expected_output.json")
	args := getTestArgs(dirPath, outFile)

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

func getTestArgs(dirPath, outFile string) common.InArgs {
	args := common.InArgs{}
	emptyStr := ""
	args.DirPath = &dirPath
	args.CommitID = &emptyStr
	args.GitBranch = &emptyStr
	args.GitURL = &emptyStr
	args.OutputFile = &outFile
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
		if line_expected != line_actual {
			fmt.Printf("Gap in line %d: expected: %s, actual: %s", i, line_expected, line_actual)
			return false, nil
		}
	}
	return true, nil
}
