package cmd

import (
	"github.com/Conflux-Chain/go-conflux-util/api"
	nhApi "github.com/Conflux-Chain/neurahive-scan/api"
	"github.com/spf13/cobra"
)

var (
	apiCmd = &cobra.Command{
		Use:   "api",
		Short: "run rest api server",
		Run:   startApiService,
	}
)

func init() {
	rootCmd.AddCommand(apiCmd)
}

func startApiService(*cobra.Command, []string) {
	dataCtx := MustInitDataContext()
	defer dataCtx.Close()

	nhApi.Init(dataCtx.Eth, dataCtx.DB)

	api.MustServeFromViper(nhApi.RegisterRouter)
}
