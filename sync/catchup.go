package sync

import (
	"context"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/Conflux-Chain/neurahive-scan/store"
	"github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go"
	"github.com/openweb3/web3go/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"math/big"
	"strings"
	"time"
)

type CatchupSyncer struct {
	conf           *SyncConfig
	sdk            *web3go.Client
	db             *store.MysqlStore
	currentBlock   uint64
	finalizedBlock uint64
	flowAddr       common.Address
	flowSubmitSig  common.Hash
}

func MustNewCatchupSyncer(sdk *web3go.Client, db *store.MysqlStore, conf SyncConfig) *CatchupSyncer {
	var flow struct {
		Address              string
		SubmitEventSignature string
	}
	viperutil.MustUnmarshalKey("flow", &flow)

	return &CatchupSyncer{
		conf:          &conf,
		sdk:           sdk,
		db:            db,
		flowAddr:      common.HexToAddress(flow.Address),
		flowSubmitSig: common.HexToHash(flow.SubmitEventSignature),
	}
}

func (s *CatchupSyncer) Sync(ctx context.Context) {
	logrus.Info("Catchup syncer starting to sync eth data")
	for {
		needProcess := s.tryBlockRange()
		if !needProcess || s.interrupted(ctx) {
			return
		}

		if err := s.syncRange(ctx, s.currentBlock, s.finalizedBlock); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"currentBlock":   s.currentBlock,
				"finalizedBlock": s.finalizedBlock,
			}).Error("Catchup syncer sync range")
			time.Sleep(time.Second)
			continue
		}
	}
}

func (s *CatchupSyncer) syncRange(ctx context.Context, rangeStart, rangeEnd uint64) error {
	start, end := s.nextSyncRange(rangeStart, rangeEnd)

	for {
		var logs []types.Log
		var err error
		logs, start, end, err = s.queryFlowSubmitsBestEffort(s.sdk, start, end, s.flowAddr, s.flowSubmitSig)
		if err != nil {
			return err
		}

		var bn2TimeMap map[uint64]uint64
		var rangeEndBlock *types.Block
		if len(logs) > 0 {
			blockNums := make([]types.BlockNumber, 0)
			for _, log := range logs {
				blockNums = append(blockNums, types.BlockNumber(log.BlockNumber))
			}
			bn2TimeMap, err = mapBlockNum2Time(ctx, s.sdk, blockNums, s.conf.BatchBlocksOnBatchCall)
		} else {
			rangeEndBlock, err = s.sdk.Eth.BlockByNumber(types.BlockNumber(end), false)
		}
		if err != nil {
			return err
		}

		block := s.convertBlock(logs, bn2TimeMap, rangeEndBlock)
		txs, err := s.convertTxs(ctx, logs, bn2TimeMap)
		if err != nil {
			return err
		}
		submits, err := s.convertSubmits(logs, bn2TimeMap)
		if err != nil {
			return err
		}
		err = s.db.Push(block, txs, submits)
		if err != nil {
			return err
		}

		if end >= rangeEnd || s.interrupted(ctx) {
			break
		}

		start, end = s.nextSyncRange(end+1, rangeEnd)
	}

	return nil
}

func (s *CatchupSyncer) nextSyncRange(curStart, rangeEnd uint64) (uint64, uint64) {
	start, end := curStart, rangeEnd

	if s.conf.BatchBlocksOnCatchup > 0 {
		end = start + s.conf.BatchBlocksOnCatchup - 1
		if end > rangeEnd {
			end = rangeEnd
		}
	}

	return start, end
}

// queryFlowSubmitsBestEffort queries flow submits from the contract event logs between a block range of (bnFrom, bnTo).
// It returns the logs, the actual start and end block numbers of the queried range, and an error if any.
// It may not return all the data for the whole range, but only the available or feasible data.
// It uses a binary search algorithm to find the optimal range that maximizes the number of logs returned.
func (s *CatchupSyncer) queryFlowSubmitsBestEffort(w3c *web3go.Client, bnFrom, bnTo uint64, flowAddr common.Address, flowSubmitSig common.Hash) ([]types.Log, uint64, uint64, error) {
	start, end := bnFrom, bnTo

	for {
		logs, err := queryFlowSubmits(w3c, start, end, flowAddr, flowSubmitSig)
		if err == nil {
			return logs, start, end, nil
		}

		if strings.Contains(err.Error(), "please narrow down your filter condition") {
			end = start + (end-start)/2
			continue
		}

		return nil, 0, 0, err
	}
}

func (s *CatchupSyncer) tryBlockRange() bool {
	for try := 1; ; try++ {
		err := s.updateBlockRange()
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"try":      try,
				"curBlk":   s.currentBlock,
				"finalBlk": s.finalizedBlock}).Info("try block range")
			time.Sleep(time.Second)
			continue
		}

		return s.currentBlock <= s.finalizedBlock
	}
}

func (s *CatchupSyncer) updateBlockRange() error {
	maxBlock, ok, err := s.db.MaxBlock()
	if err != nil {
		return errors.WithMessage(err, "failed to get max block from db")
	}
	if ok {
		s.currentBlock = maxBlock + 1
	} else {
		s.currentBlock = s.conf.BlockWhenFlowCreated
	}

	finalizedBlock, err := s.sdk.Eth.BlockByNumber(types.FinalizedBlockNumber, false)
	if err != nil {
		return errors.WithMessage(err, "failed to get finalized block")
	}
	s.finalizedBlock = finalizedBlock.Number.Uint64()

	return nil
}

func (s *CatchupSyncer) convertBlock(logs []types.Log, blockNum2TimeMap map[uint64]uint64, endBlk *types.Block) *store.Block {
	var block *store.Block

	if len(logs) > 0 {
		log := (logs)[len(logs)-1]
		ts := blockNum2TimeMap[log.BlockNumber]
		blk := &types.Block{
			Number:    new(big.Int).SetUint64(log.BlockNumber),
			Hash:      log.BlockHash,
			Timestamp: ts,
		}
		block = store.NewBlock(blk)
	} else {
		block = store.NewBlock(endBlk)
	}

	return block
}

func (s *CatchupSyncer) convertTxs(ctx context.Context, logs []types.Log, blockNum2TimeMap map[uint64]uint64) ([]*store.Tx, error) {
	var txns []*store.Tx
	if len(logs) == 0 {
		return txns, nil
	}

	txHashes := make([]common.Hash, 0)
	for _, log := range logs {
		txHashes = append(txHashes, log.TxHash)
	}

	txs, err := queryTxsByHashes(ctx, s.sdk, txHashes, s.conf.BatchTxsOnBatchCall)
	if err != nil {
		return nil, err
	}

	for _, tx := range txs {
		ts := blockNum2TimeMap[tx.BlockNumber.Uint64()]
		blockTime := time.Unix(int64(ts), 0)
		txn, err := s.convertTx(&blockTime, tx)
		if err != nil {
			return nil, err
		}
		txns = append(txns, txn)
	}

	return txns, nil
}

func (s *CatchupSyncer) convertTx(blkTime *time.Time, txn *types.TransactionDetail) (*store.Tx, error) {
	tx := store.NewTx(blkTime, txn)
	fromId, err := s.db.AddressStore.Add(nil, tx.From, blkTime)
	if err != nil {
		return nil, err
	}
	toId, err := s.db.AddressStore.Add(nil, tx.To, blkTime)
	if err != nil {
		return nil, err
	}
	tx.FromId = fromId
	tx.ToId = toId
	return tx, nil
}

func (s *CatchupSyncer) convertSubmits(logs []types.Log, blockNum2TimeMap map[uint64]uint64) ([]*store.Submit, error) {
	var submits []*store.Submit

	for _, log := range logs {
		ts := blockNum2TimeMap[log.BlockNumber]
		blockTime := time.Unix(int64(ts), 0)
		submit, err := store.NewSubmit(&blockTime, &log)
		if err != nil {
			return nil, err
		}

		senderId, err := s.db.AddressStore.Add(nil, submit.Sender, &blockTime)
		if err != nil {
			return nil, err
		}

		submit.SenderId = senderId
		submits = append(submits, submit)
	}

	return submits, nil
}

func (s *CatchupSyncer) interrupted(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}
