package main

import (
	"fmt"
	"os"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
)

func main() {
	var inArgs common.InArgs
	err := common.ParseInArgs(&inArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing arguments: %v. exiting...\n", err)
		os.Exit(1)
	}

	err = controller.Start(inArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running topology analysis: %v. exiting...", err)
		os.Exit(1)
	}
}
