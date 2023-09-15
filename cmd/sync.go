package cmd

import (
	"context"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	nhSync "github.com/Conflux-Chain/neurahive-scan/sync"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	syncCtx := MustInitSyncContext()
	defer syncCtx.Close()

	var conf nhSync.SyncConfig
	viperutil.MustUnmarshalKey("sync", &conf)

	catchupSyncer := nhSync.MustNewCatchupSyncer(syncCtx.Eth, syncCtx.DB, conf)
	syncer := nhSync.MustNewSyncer(syncCtx.Eth, syncCtx.DB, conf, catchupSyncer)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	go syncer.Sync(ctx, &wg)

	GracefulShutdown(&wg, cancel)
}

func GracefulShutdown(wg *sync.WaitGroup, cancel context.CancelFunc) {
	// Handle sigterm and await termChan signal
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGTERM, syscall.SIGINT)

	// Wait for SIGTERM to be captured
	<-termChan
	logrus.Info("SIGTERM/SIGINT received, shutdown process initiated")

	// Cancel to notify active goroutines to clean up.
	cancel()

	logrus.Info("Waiting for shutdown...")
	wg.Wait()

	logrus.Info("Shutdown gracefully")
}
