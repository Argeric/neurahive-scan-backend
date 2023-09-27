package cmd

import (
	"context"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	nhSync "github.com/Conflux-Chain/neurahive-scan/sync"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sync"
)

var (
	syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Start sync, including block/submitLog",
		Run:   startSyncService,
	}
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

func startSyncService(*cobra.Command, []string) {
	logrus.Info("Start to sync evm space blockchain data into database")
	dataCtx := MustInitDataContext()
	defer dataCtx.Close()

	var conf nhSync.SyncConfig
	viperutil.MustUnmarshalKey("sync", &conf)

	catchupSyncer := nhSync.MustNewCatchupSyncer(dataCtx.Eth, dataCtx.DB, conf)
	syncer := nhSync.MustNewSyncer(dataCtx.Eth, dataCtx.DB, conf, catchupSyncer)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	go syncer.Sync(ctx, &wg)

	GracefulShutdown(&wg, cancel)
}
