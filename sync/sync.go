package sync

import (
	"context"
	"github.com/Conflux-Chain/neurahive-scan/store"
	"github.com/openweb3/web3go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type SyncConfig struct {
	BlockWhenFlowCreated  uint64
	SkipBlocksAheadLatest uint64 `default:"30"`
}

type Syncer struct {
	conf                *SyncConfig
	sdk                 *web3go.Client
	db                  *store.MysqlStore
	currentBlock        uint64
	syncIntervalNormal  time.Duration
	syncIntervalCatchUp time.Duration
}

// MustNewSyncer creates an instance of Syncer to sync blockchain data.
func MustNewSyncer(sdk *web3go.Client, db *store.MysqlStore, conf SyncConfig) *Syncer {
	syncer := &Syncer{
		conf:                &conf,
		sdk:                 sdk,
		db:                  db,
		syncIntervalNormal:  time.Second,
		syncIntervalCatchUp: time.Millisecond,
	}

	// Load last sync block information
	syncer.mustLoadLastSyncBlock()

	return syncer
}

// Load last sync block from database to continue synchronization.
func (s *Syncer) mustLoadLastSyncBlock() {
	loaded, err := s.loadLastSyncBlock()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load last sync block from db")
	}

	// Load db sync start block config on initial loading if necessary.
	if !loaded && s.conf != nil {
		s.currentBlock = s.conf.BlockWhenFlowCreated
	}
}

func (s *Syncer) loadLastSyncBlock() (loaded bool, err error) {
	maxBlock, ok, err := s.db.MaxBlock()
	if err != nil {
		return false, errors.WithMessage(err, "failed to get max block from block table")
	}

	if ok {
		s.currentBlock = maxBlock + 1
	}

	return ok, nil
}

func (s *Syncer) Sync(ctx context.Context, wg *sync.WaitGroup) {
	logrus.Info("Syncer starting to sync eth data")

	wg.Add(1)
	defer wg.Done()

	ticker := time.NewTicker(s.syncIntervalCatchUp)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("DB syncer shutdown ok")
			return
		case <-ticker.C:
			if err := s.doTicker(ticker); err != nil {
				logrus.WithError(err).
					WithField("currentBlock", s.currentBlock).
					Warn("Syncer failed to sync eth data")
			}
		}
	}
}

func (s *Syncer) doTicker(ticker *time.Ticker) error {
	logrus.Debug("Syncer ticking")

	complete, err := s.syncOnce()

	if err != nil {
		ticker.Reset(s.syncIntervalNormal)
		return err
	} else if complete {
		ticker.Reset(s.syncIntervalNormal)
	} else {
		ticker.Reset(s.syncIntervalCatchUp)
	}

	return nil
}

func (s *Syncer) syncOnce() (bool, error) {
	// get latest block
	latestBlock, err := s.sdk.Eth.BlockNumber()
	if err != nil {
		return false, err
	}

	// check latest block
	curBlock := s.currentBlock
	if curBlock > latestBlock.Uint64()-s.conf.SkipBlocksAheadLatest {
		return true, nil
	}

	// get eth data
	data, err := store.QueryEthData(s.sdk, curBlock)
	if err != nil {
		return false, err
	}
	if data == nil {
		return true, nil
	}

	// check pivot hash
	latestBlockHash, err := s.getStoreLatestBlockHash()
	if err != nil {
		return false, err
	}
	if len(latestBlockHash) > 0 && data.Block.ParentHash.Hex()[2:] != latestBlockHash {
		latestStoreBlock := s.latestStoreBlock()
		if err := s.pivotSwitchRevert(latestStoreBlock); err != nil {
			return false, err
		}
		return false, nil
	}

	// persist db
	if err = s.db.Push(data); err != nil {
		return false, err
	}

	// increase currentBlock
	s.currentBlock += 1

	return false, nil
}

func (s *Syncer) getStoreLatestBlockHash() (string, error) {
	blockFlowContractCreated := s.conf.BlockWhenFlowCreated
	if s.currentBlock <= blockFlowContractCreated {
		return "", nil
	}

	latestBlockNo := s.latestStoreBlock()
	hash, _, err := s.db.BlockHash(latestBlockNo)
	return hash, err
}

func (s *Syncer) latestStoreBlock() uint64 {
	if s.currentBlock > 0 {
		return s.currentBlock - 1
	}

	return 0
}

func (s *Syncer) pivotSwitchRevert(revertBlock uint64) error {
	// check
	logger := logrus.WithFields(logrus.Fields{
		"revertBlock":      revertBlock,
		"latestStoreBlock": s.latestStoreBlock(),
	})
	if revertBlock == 0 {
		return errors.New("genesis block must not be reverted")
	}
	if revertBlock >= s.currentBlock {
		return nil
	}

	// pop from db
	logger.Info("[Syncer]revert eth data at block %v", revertBlock)
	if err := s.db.Pop(revertBlock); err != nil {
		return errors.WithMessage(err, "failed to pop eth data from db")
	}

	// update currentBlock
	s.currentBlock = revertBlock

	return nil
}
