package main

import (
	"flag"
	"fmt"
)

type InArgs struct {
	DirPath      *string
	OutputFile   *string
	SynthNetpols *bool
	Quiet        *bool
	Verbose      *bool
}

func ParseInArgs(args *InArgs) error {
	args.DirPath = flag.String("dirpath", "", "input directory path")
	args.OutputFile = flag.String("outputfile", "", "file path to store results")
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

	return nil
}
