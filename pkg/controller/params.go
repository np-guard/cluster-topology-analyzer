package controller

import (
	"flag"
	"fmt"
	"go.uber.org/zap"
)

func ParseInArgs(args *InArgs) error {
	args.DirPath = flag.String("dirpath", "", "input directory path")
	args.GitURL = flag.String("giturl", "", "git repository url")
	args.GitBranch = flag.String("gitbranch", "", "git repository branch")
	args.CommitID = flag.String("commitid", "", "gitsecure run id")
	args.OutputFile = flag.String("outputfile", "", "file path to store results")
	args.SynthNetpols = flag.Bool("netpols", false, "Whether to synthesize NetworkPolicies out of the discovered connections")
	flag.Parse()

	if *args.DirPath == "" ||
		*args.GitBranch == "" ||
		*args.CommitID == "" ||
		*args.GitURL == "" {
		zap.S().Debugf("missing parameters: [%s %s %s %s]", *args.DirPath, *args.GitURL, *args.GitBranch, *args.CommitID)
		flag.PrintDefaults()
		return fmt.Errorf("missing params")
	}

	return nil
}
