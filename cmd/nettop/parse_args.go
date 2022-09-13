package main

import (
	"flag"
	"fmt"
)

type InArgs struct {
	DirPath      *string
	OutputFile   *string
	SynthNetpols *bool
}

func ParseInArgs(args *InArgs) error {
	args.DirPath = flag.String("dirpath", "", "input directory path")
	args.OutputFile = flag.String("outputfile", "", "file path to store results")
	args.SynthNetpols = flag.Bool("netpols", false, "Whether to synthesize NetworkPolicies out of the discovered connections")
	flag.Parse()

	if *args.DirPath == "" {
		flag.PrintDefaults()
		return fmt.Errorf("missing parameter: %s", *args.DirPath)
	}

	return nil
}
