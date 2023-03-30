package main

import (
	"flag"
	"fmt"

	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
)

type PathList []string

func (dp *PathList) String() string {
	return fmt.Sprintln(*dp)
}

func (dp *PathList) Set(path string) error {
	*dp = append(*dp, path)
	return nil
}

const (
	JSONFormat = "json"
	YamlFormat = "yaml"
)

type InArgs struct {
	DirPaths     PathList
	OutputFile   *string
	OutputFormat *string
	DNSPort      *int
	SynthNetpols *bool
	Quiet        *bool
	Verbose      *bool
}

func ParseInArgs() (*InArgs, error) {
	args := InArgs{}
	flag.Var(&args.DirPaths, "dirpath", "input directory path")
	args.OutputFile = flag.String("outputfile", "", "file path to store results")
	args.OutputFormat = flag.String("format", JSONFormat, "output format; must be either \"json\" or \"yaml\"")
	args.SynthNetpols = flag.Bool("netpols", false, "whether to synthesize NetworkPolicies to allow only the discovered connections")
	args.DNSPort = flag.Int("dnsport", controller.DefaultDNSPort, "specify DNS port to be used in egress rules of synthesized NetworkPolicies")
	args.Quiet = flag.Bool("q", false, "runs quietly, reports only severe errors and results")
	args.Verbose = flag.Bool("v", false, "runs with more informative messages printed to log")
	flag.Parse()

	if len(args.DirPaths) == 0 {
		flag.PrintDefaults()
		return nil, fmt.Errorf("missing parameter: dirpath")
	}
	if *args.Quiet && *args.Verbose {
		flag.PrintDefaults()
		return nil, fmt.Errorf("-q and -v cannot be specified together")
	}
	if *args.OutputFormat != JSONFormat && *args.OutputFormat != YamlFormat {
		flag.PrintDefaults()
		return nil, fmt.Errorf("wrong output format %s; must be either json or yaml", *args.OutputFormat)
	}

	return &args, nil
}
