package main

import (
	"encoding/json"
	"fmt"
	"os"

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

func writeContent(outputFile string, content interface{}) error {
	const indent = "    "

	buf, err := json.MarshalIndent(content, "", indent)
	if err != nil {
		return err
	}

	if outputFile != "" {
		return writeBufToFile(outputFile, buf)
	}

	fmt.Printf("connection topology reports: \n ---\n%s\n---", string(buf))
	return nil
}

// Based on the arguments it is given, scans all YAML files,
// detects all required connection between resources and outputs a json connectivity report
// (or NetworkPolicies to allow only this connectivity)
func detectTopology(args InArgs) error {
	logger := controller.NewDefaultLogger()
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

	if err := writeContent(*args.OutputFile, content); err != nil {
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
