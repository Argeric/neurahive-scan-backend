package cmd

import (
	"context"
	nhSync "github.com/Argeric/neurahive-scan-backend/sync"
	"github.com/Argeric/neurahive-scan-backend/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sync"
)

var (
	syncOpt struct {
		epochFrom, epochTo uint64
	}

	syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Start sync sync, including epoch/block/tx/submitLog",
		Run:   startSyncService,
	}
)

func init() {
	syncCmd.Flags().Uint64Var(
		&syncOpt.epochFrom, "start", 0,
		"the epoch from which sync will start",
	)

	rootCmd.AddCommand(syncCmd)
}

func startSyncService(*cobra.Command, []string) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	syncCtx := util.MustInitSyncContext()
	defer syncCtx.Close()

	startSyncCfxDatabase(ctx, &wg, syncCtx)

	util.GracefulShutdown(&wg, cancel)
}

func startSyncCfxDatabase(ctx context.Context, wg *sync.WaitGroup, syncCtx util.SyncContext) {
	logrus.Info("Start to sync core space blockchain data into database")

	syncer := nhSync.MustNewEpochSyncer(syncCtx.Cfx, syncCtx.DB)
	go syncer.Sync(ctx, wg)
}
