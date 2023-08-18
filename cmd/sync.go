package cmd

import (
	"context"
	"github.com/Argeric/neurahive-scan-backend/util"
	"github.com/spf13/cobra"
	"sync"
)

var (
	syncOpt struct {
		epochFrom, epochTo uint64
	}

	syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Start sync service, including epoch/block/tx/submitLog",
		Run:   startSyncService,
	}
)

func init() {
	syncCmd.Flags().Uint64Var(
		&syncOpt.epochFrom, "start", 0,
		"the epoch from which sync will start",
	)
}

func startSyncService(command *cobra.Command, []string) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	storeCtx := util.MustInitStoreContext()
	defer storeCtx.Close()

	syncCtx := util.MustInitSyncContext(storeCtx)
	defer syncCtx.Close()

	if syncOpt.dbSyncEnabled { // start DB sync
		syncer := startSyncCfxDatabase(ctx, &wg, syncCtx)
		subs = append(subs, syncer)
	}

	util.GracefulShutdown(&wg, cancel)
}
