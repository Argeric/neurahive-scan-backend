package cmd

import (
	"context"
	"github.com/Conflux-Chain/neurahive-scan/stat"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sync"
)

var (
	statCmd = &cobra.Command{
		Use:   "stat",
		Short: "Start stat, including transactions and data size of storage",
		Run:   startStatService,
	}
)

func init() {
	rootCmd.AddCommand(statCmd)
}

func startStatService(*cobra.Command, []string) {
	logrus.Info("Start to stat transactions and data size of storage")
	dataCtx := MustInitDataContext()
	defer dataCtx.Close()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	startTime := stat.MustDefaultRangeStart(dataCtx.Eth)
	st := stat.MustNewTxStat(dataCtx.Eth, dataCtx.DB, startTime)
	go st.DoStat(ctx, &wg)

	GracefulShutdown(&wg, cancel)
}
