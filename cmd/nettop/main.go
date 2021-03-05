package main

import (
	"os"

	"github.ibm.com/gitsecure-net-top/pkg/common"
	"github.ibm.com/gitsecure-net-top/pkg/controller"
	"go.uber.org/zap"
)

func main() {
	logger := common.SetupLogger()
	defer logger.Sync()

	var inArgs common.InArgs
	if err := common.ParseInArgs(&inArgs); err != nil {
		zap.S().Debug("error parsing arguments, exiting...")
		os.Exit(1)
	}

	controller.Start(inArgs)
}
