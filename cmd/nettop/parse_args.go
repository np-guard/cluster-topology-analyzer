package main

import (
	"flag"
	"fmt"
)

const (
	JSONFormat = "json"
	YamlFormat = "yaml"
)

type InArgs struct {
	DirPath      *string
	OutputFile   *string
	OutputFormat *string
	SynthNetpols *bool
	Quiet        *bool
	Verbose      *bool
}

func ParseInArgs(args *InArgs) error {
	args.DirPath = flag.String("dirpath", "", "input directory path")
	args.OutputFile = flag.String("outputfile", "", "file path to store results")
	args.OutputFormat = flag.String("format", JSONFormat, "output format (must be either json or yaml)")
	args.SynthNetpols = flag.Bool("netpols", false, "whether to synthesize NetworkPolicies to allow only the discovered connections")
	args.Quiet = flag.Bool("q", false, "runs quietly, reports only severe errors and results")
	args.Verbose = flag.Bool("v", false, "runs with more informative messages printed to log")
	flag.Parse()

	if *args.DirPath == "" {
		flag.PrintDefaults()
		return fmt.Errorf("missing parameter: %s", *args.DirPath)
	}
	if *args.Quiet && *args.Verbose {
		flag.PrintDefaults()
		return fmt.Errorf("-q and -v cannot be specified together")
	}
	if *args.OutputFormat != JSONFormat && *args.OutputFormat != YamlFormat {
		flag.PrintDefaults()
		return fmt.Errorf("wrong output format %s; must be either json or yaml", *args.OutputFormat)
	}

	return nil
}
