package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
)

func writeBufToFile(filepath string, buf []byte) error {
	fp, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", filepath, err)
	}
	_, err = fp.Write(buf)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", filepath, err)
	}
	fp.Close()
	return nil
}

func yamlMarshalUsingJSON(content interface{}) ([]byte, error) {
	// Directly marshaling content into YAML, results in malformed Kubernetes resources.
	// This is because K8s NetworkPolicy struct has json field tags, but no yaml field tags (also true for other resources).
	// The (somewhat ugly) solution is to first marshal content to json, unmarshal to an interface{} var and marshal to yaml
	buf, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	var contentFromJSON interface{}
	err = json.Unmarshal(buf, &contentFromJSON)
	if err != nil {
		return nil, err
	}

	buf, err = yaml.Marshal(contentFromJSON)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func writeContent(outputFile, outputFormat string, content interface{}) error {
	var buf []byte
	var err error
	if outputFormat == YamlFormat {
		buf, err = yamlMarshalUsingJSON(content)
	} else {
		const indent = "    "
		buf, err = json.MarshalIndent(content, "", indent)

	}
	if err != nil {
		return err
	}

	if outputFile != "" {
		return writeBufToFile(outputFile, buf)
	}

	fmt.Println(string(buf))
	return nil
}

// returns verbosity level based on the -q and -v switches
func getVerbosity(args InArgs) controller.Verbosity {
	verbosity := controller.MediumVerbosity
	if *args.Quiet {
		verbosity = controller.LowVerbosity
	} else if *args.Verbose {
		verbosity = controller.HighVerbosity
	}
	return verbosity
}

// Based on the arguments it is given, scans all YAML files,
// detects all required connection between resources and outputs a json connectivity report
// (or NetworkPolicies to allow only this connectivity)
func detectTopology(args InArgs) error {
	logger := controller.NewDefaultLoggerWithVerbosity(getVerbosity(args))
	synth := controller.NewPoliciesSynthesizer(controller.WithLogger(logger))

	var content interface{}
	if args.SynthNetpols != nil && *args.SynthNetpols {
		policies, synthesisErr := synth.PoliciesFromFolderPath(*args.DirPath)
		if synthesisErr != nil {
			logger.Errorf(synthesisErr, "error synthesizing policies")
			return synthesisErr
		}
		content = controller.NetpolListFromNetpolSlice(policies)
	} else {
		var err error
		content, err = synth.ConnectionsFromFolderPath(*args.DirPath)
		if err != nil {
			logger.Errorf(err, "error extracting connections")
			return err
		}
	}

	if err := writeContent(*args.OutputFile, *args.OutputFormat, content); err != nil {
		logger.Errorf(err, "error writing results")
		return err
	}

	return nil
}

func main() {
	var inArgs InArgs
	err := ParseInArgs(&inArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing arguments: %v. exiting...\n", err)
		os.Exit(1)
	}

	err = detectTopology(inArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running topology analysis: %v. exiting...", err)
		os.Exit(1)
	}
}
