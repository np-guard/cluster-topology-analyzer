package main

import (
	"errors"
	"os"
	"syscall"

	"go.uber.org/zap"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
	"github.com/np-guard/cluster-topology-analyzer/pkg/controller"
)

func runAnalysis() int {
	logger := common.SetupLogger()
	defer func() {
		err := logger.Sync()
		// If stderr is a TTY we might not be able to sync.
		// See https://github.com/uber-go/zap/issues/991#issuecomment-962098428 for
		// why we ignore ENOTTY.  On OSX, we must ignore EBADF.
		if err != nil && !errors.Is(err, syscall.ENOTTY) && !errors.Is(err, syscall.EBADF) {
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
