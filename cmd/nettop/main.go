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

	// c := []common.Connections{}
	// c1 := common.Connections{}
	// c1.Link = common.Service{}
	// c1.Source = common.Resource{}
	// c1.Target = common.Resource{}
	// c = append(c, c1)
	// b, _ := json.MarshalIndent(c, "", "    ")
	// zap.S().Debugf("\n%s", string(b))
	controller.Start(inArgs)
}
