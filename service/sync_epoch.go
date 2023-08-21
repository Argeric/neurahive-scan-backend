package service

import (
	"context"
	"github.com/Argeric/neurahive-scan-backend/store"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type syncConfig struct {
	Preload   uint64
	FromEpoch uint64
}

type EpochSyncer struct {
	conf                *syncConfig
	cfx                 sdk.ClientOperator
	db                  *store.MysqlStore
	epochFrom           uint64
	maxSyncEpochs       uint64
	syncIntervalNormal  time.Duration
	syncIntervalCatchUp time.Duration
}

// MustNewEpochSyncer creates an instance of DatabaseSyncer to sync blockchain data.
func MustNewEpochSyncer(cfx sdk.ClientOperator, db *store.MysqlStore) *EpochSyncer {
	var conf syncConfig
	viperutil.MustUnmarshalKey("sync", &conf)

	syncer := &EpochSyncer{
		conf:                &conf,
		cfx:                 cfx,
		db:                  db,
		epochFrom:           0,
		maxSyncEpochs:       conf.Preload,
		syncIntervalNormal:  time.Second,
		syncIntervalCatchUp: time.Millisecond,
	}

	return syncer
}

func (syncer *EpochSyncer) Sync(ctx context.Context, wg *sync.WaitGroup) {
	logrus.Info("DB sync starting to sync epoch data")

	wg.Add(1)
	defer wg.Done()

	breakLoop := false
	quit := func() {
		breakLoop = true
		logrus.Info("DB syncer shutdown ok")
	}

	ticker := time.NewTicker(syncer.syncIntervalCatchUp)
	defer ticker.Stop()

	for !breakLoop {
		select { // first class priority
		case <-ctx.Done():
			quit()
		default:
			select { // second class priority
			case <-ctx.Done():
				quit()
			case <-ticker.C:
				if err := syncer.doTicker(ticker); err != nil {
					logrus.WithError(err).
						WithField("epochFrom", syncer.epochFrom).
						Error("Db syncer failed to sync epoch data")
				}
			}
		}
	}
}

func (syncer *EpochSyncer) doTicker(ticker *time.Ticker) error {
	logrus.Debug("DB sync ticking")

	complete, err := syncer.syncOnce()

	if err != nil {
		ticker.Reset(syncer.syncIntervalNormal)
		return err
	} else if complete {
		ticker.Reset(syncer.syncIntervalNormal)
	} else {
		ticker.Reset(syncer.syncIntervalCatchUp)
	}

	return nil
}

func (syncer *EpochSyncer) syncOnce() (bool, error) {

}
