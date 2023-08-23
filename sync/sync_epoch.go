package sync

import (
	"context"
	"github.com/Argeric/neurahive-scan-backend/store/blockchain"
	"github.com/Argeric/neurahive-scan-backend/store/db"
	"github.com/Argeric/neurahive-scan-backend/util"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type syncConfig struct {
	Preload   uint64
	FromEpoch uint64
	UseBatch  bool
}

type EpochSyncer struct {
	conf                *syncConfig
	cfx                 sdk.ClientOperator
	db                  *db.MysqlStore
	epochFrom           uint64
	maxSyncEpochs       uint64
	syncIntervalNormal  time.Duration
	syncIntervalCatchUp time.Duration
}

// MustNewEpochSyncer creates an instance of DatabaseSyncer to sync blockchain data.
func MustNewEpochSyncer(cfx sdk.ClientOperator, db *db.MysqlStore) *EpochSyncer {
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

	// Load last sync epoch information
	syncer.mustLoadLastSyncEpoch()

	return syncer
}

// Load last sync epoch from databse to continue synchronization.
func (syncer *EpochSyncer) mustLoadLastSyncEpoch() {
	loaded, err := syncer.loadLastSyncEpoch()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load last sync epoch range from db")
	}

	// Load db sync start epoch config on initial loading if necessary.
	if !loaded && syncer.conf != nil {
		syncer.epochFrom = syncer.conf.FromEpoch
	}
}

func (syncer *EpochSyncer) loadLastSyncEpoch() (loaded bool, err error) {
	maxEpoch, ok, err := syncer.db.MaxEpoch()
	if err != nil {
		return false, errors.WithMessage(err, "failed to get max epoch from epoch table")
	}

	if ok {
		syncer.epochFrom = maxEpoch + 1
	}

	return ok, nil
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
	logger := logrus.WithField("epochFrom", syncer.epochFrom)

	// get latest state epoch
	epoch, err := syncer.cfx.GetEpochNumber(types.EpochLatestState)
	if err != nil {
		logger.Debug("Db syncer skipped due to getting latest state failure")
		return false, errors.WithMessage(
			err, "failed to query the latest state epoch number",
		)
	}

	// check latest state epoch
	maxEpochTo := epoch.ToInt().Uint64()
	if syncer.epochFrom > maxEpochTo {
		logrus.Debug("Db syncer skipped due to already catch-up")
		return true, nil
	}

	// get epoch data
	logger.Debug("DB sync started to sync with epoch range")
	data, err := blockchain.QueryEpochData(syncer.cfx, syncer.epochFrom, syncer.conf.UseBatch)
	logrus.WithField("epoch", data.Number).Infof("data: %v", data)
	if errors.Is(err, util.ErrEpochPivotSwitched) {
		logger.WithError(err).Info("Db syncer failed to query epoch data due to pivot switch")
		return false, errors.WithMessagef(err, "failed to query epoch due to pivot switch at epoch %v", syncer.epochFrom)
	}
	if err != nil {
		return false, errors.WithMessagef(err, "failed to query epoch data for epoch %v", syncer.epochFrom)
	}

	// check pivot hash
	latestPivotHash, err := syncer.getStoreLatestPivotHash()
	if err != nil {
		logger.WithError(err).Error("Db syncer failed to get latest pivot hash from db for parent hash check")
		return false, errors.WithMessage(err, "failed to get latest pivot hash")
	}
	if len(latestPivotHash) > 0 && data.GetPivotBlock().ParentHash != latestPivotHash {
		latestStoreEpochNo := syncer.latestStoreEpoch()
		logger.WithFields(logrus.Fields{
			"latestStoreEpoch": latestStoreEpochNo,
			"latestPivotHash":  latestPivotHash,
		}).Warn("Db syncer popping latest epoch from db store due to parent hash mismatched")
		if err := syncer.pivotSwitchRevert(latestStoreEpochNo); err != nil {
			logger.WithError(err).Error(
				"Db syncer failed to pop latest epoch from db store due to parent hash mismatched",
			)
			return false, errors.WithMessage(
				err, "failed to pop latest epoch from db store due to parent hash mismatched",
			)
		}
		return false, nil
	}

	// persist db
	if err = syncer.db.Push(&data); err != nil {
		logger.WithError(err).Error("Db syncer failed to save epoch data to db")
		return false, errors.WithMessage(err, "failed to save epoch data to db")
	}

	// increase epochFrom
	syncer.epochFrom += 1

	return true, nil
}

func (syncer *EpochSyncer) getStoreLatestPivotHash() (types.Hash, error) {
	if syncer.epochFrom == 0 { // no epoch synchronized yet
		return "", nil
	}

	latestEpochNo := syncer.latestStoreEpoch()
	pivotHash, _, err := syncer.db.PivotHash(latestEpochNo)
	return types.Hash(pivotHash), err
}

func (syncer *EpochSyncer) latestStoreEpoch() uint64 {
	if syncer.epochFrom > 0 {
		return syncer.epochFrom - 1
	}

	return 0
}

func (syncer *EpochSyncer) pivotSwitchRevert(revertTo uint64) error {
	// check
	logger := logrus.WithFields(logrus.Fields{
		"revertToEpoch":    revertTo,
		"latestStoreEpoch": syncer.latestStoreEpoch(),
	})
	if revertTo == 0 {
		logger.Debug("Db syncer skipped pivot switch revert due to genesis epoch cannot revert")
		return errors.New("genesis epoch must not be reverted")
	}
	if revertTo >= syncer.epochFrom {
		logger.Debug("Db syncer skipped pivot switch revert due to not catched up yet")
		return nil
	}

	// pop from db
	logger.Info("Db syncer reverting epoch data due to pivot chain switch")
	if err := syncer.db.Pop(revertTo); err != nil {
		logger.WithError(err).Error(
			"Db syncer failed to pop epoch data from db due to pivot switch",
		)
		return errors.WithMessage(err, "failed to pop epoch data from db")
	}

	// update epochFrom
	syncer.epochFrom = revertTo

	return nil
}
