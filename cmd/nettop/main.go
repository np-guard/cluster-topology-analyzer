package main

import (
	"os"

	"go.uber.org/zap"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
)

func runAnalysis() int {
	logger := common.SetupLogger()
	defer func() {
		err := logger.Sync()
		if err != nil {
			panic(err)
		}
	}()

	var inArgs common.InArgs
	err := common.ParseInArgs(&inArgs)
	if err != nil {
		zap.S().Debug("error parsing arguments, exiting...")
		return 1
	}

	err = controller.Start(inArgs)
	if err != nil {
		zap.S().Debug("error running topology analysis exiting...")
		return 1
	}
	return 0
}

func main() {
	os.Exit(runAnalysis())
}
