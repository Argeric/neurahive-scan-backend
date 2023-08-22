package blockchain

import (
	"fmt"
	"github.com/Argeric/neurahive-scan-backend/util"
	"github.com/Argeric/neurahive-scan-backend/util/blacklist"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	sdkerr "github.com/Conflux-Chain/go-conflux-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	emptyEpochData = EpochData{}

	errBlockValidationFailed      = errors.New("epoch block validation failed")
	errTxsReceiptValidationFailed = errors.New("transaction receipt validation failed")
)

type EpochData struct {
	Number   uint64
	Blocks   []*types.Block
	Receipts map[types.Hash]*types.TransactionReceipt
}

func (epoch *EpochData) GetPivotBlock() *types.Block {
	return epoch.Blocks[len(epoch.Blocks)-1]
}

func QueryEpochData(cfx sdk.ClientOperator, epochNumber uint64, useBatch bool) (EpochData, error) {
	// get block hashes.
	epoch := types.NewEpochNumberUint64(epochNumber)
	blockHashes, err := cfx.GetBlocksByEpoch(epoch)
	if err == nil && len(blockHashes) == 0 {
		err = errors.New("invalid epoch data (must have at least one block)")
	}
	if err != nil {
		return emptyEpochData, errors.WithMessagef(err, "failed to get blocks by epoch %v", epochNumber)
	}

	// get blocks
	pivotHash := blockHashes[len(blockHashes)-1]
	blocks := make([]*types.Block, 0, len(blockHashes))
	logger := logrus.WithFields(logrus.Fields{
		"epochNo": epochNumber, "pivotHash": pivotHash,
	})
	anyBlockExecuted := false
	for _, hash := range blockHashes {
		block, err := cfx.GetBlockByHashWithPivotAssumption(hash, pivotHash, hexutil.Uint64(epochNumber))
		if err == nil {
			err = validateBlock(&block, epochNumber, hash)
		}
		if checkPivotSwitchWithError(err) {
			logger.WithFields(logrus.Fields{
				"blockHash":   hash,
				"blockHeader": &(block.BlockHeader),
			}).WithError(err).Info(
				"Failed to get block by hash with pivot assumption (regarded as pivot switch)",
			)
			err = util.ErrEpochPivotSwitched
		}
		if err != nil {
			return emptyEpochData, errors.WithMessagef(err, "failed to get block by hash %v", hash)
		}

		anyBlockExecuted = anyBlockExecuted || !util.IsEmptyBlock(&block)
		blocks = append(blocks, &block)
	}

	// get receipts
	var epochReceipts [][]types.TransactionReceipt
	if anyBlockExecuted && useBatch {
		epochReceipts, err = cfx.GetEpochReceiptsByPivotBlockHash(pivotHash)
		if checkPivotSwitchWithError(err) {
			logger.WithError(err).Info(
				"Failed to get epoch receipts with pivot assumption (regarded as pivot switch)",
			)
			err = util.ErrEpochPivotSwitched
		}
		if err != nil {
			return emptyEpochData, errors.WithMessagef(
				err, "failed to get epoch receipts by pivot %v", pivotHash,
			)
		}
	}

	// build receipts
	receipts := make(map[types.Hash]*types.TransactionReceipt)
	for i, block := range blocks {
		var logIndex uint64
		for j, tx := range block.Transactions {
			logger := logrus.WithFields(logrus.Fields{
				"i": i, "j": j,
				"epoch": epochNumber,
				"block": block.Hash.String(),
				"tx":    tx.Hash.String(),
			})
			if !util.IsTxExecutedInBlock(&tx) {
				logger.Debug("Transaction not executed in block")
				continue
			}
			var receipt *types.TransactionReceipt
			if useBatch {
				if epochReceipts == nil {
					logger.Info("Failed to match tx receipts due to epoch receipts nil (regarded as pivot switch)")
					return emptyEpochData, errors.WithMessage(util.ErrEpochPivotSwitched, "batch retrieved epoch receipts nil")
				}
				if i >= len(epochReceipts) || j >= len(epochReceipts[i]) {
					logger.WithField("epochReceipts", epochReceipts).Error("Batch retrieved receipts out of bound")
					return emptyEpochData, errors.New("batch retrieved receipts out of bound")
				}
				receipt = &epochReceipts[i][j]
				if receipt == nil {
					logger.Error("Batch retrieved receipt not found")
					return emptyEpochData, errors.Errorf("batch retrieved receipt not found for tx %v", tx.Hash)
				}
			} else {
				receipt, err = cfx.GetTransactionReceipt(tx.Hash)
				if err != nil {
					return emptyEpochData, errors.WithMessagef(err, "Failed to get receipt by tx hash %v", tx.Hash)
				}
				if receipt == nil {
					logger.Info("Failed to get tx receipt due to receipt nil (regarded as pivot switch)")
					return emptyEpochData, errors.WithMessage(util.ErrEpochPivotSwitched, "retrieved tx receipt nil")
				}
			}
			if err := validateTxsReceipt(receipt, epochNumber, block, &tx); err != nil {
				logger.WithError(err).Info("Failed to get transaction receipt (regarded as pivot switch)")
				return emptyEpochData, errors.WithMessage(util.ErrEpochPivotSwitched, err.Error())
			}
			var txLogIndex uint64
			logs := make([]types.Log, 0, len(receipt.Logs))
			for _, log := range receipt.Logs {
				log.BlockHash = &receipt.BlockHash
				log.EpochNumber = types.NewBigInt(uint64(*receipt.EpochNumber))
				log.TransactionHash = &receipt.TransactionHash
				log.TransactionIndex = types.NewBigInt(uint64(receipt.Index))
				log.LogIndex = types.NewBigInt(logIndex)
				log.TransactionLogIndex = types.NewBigInt(txLogIndex)
				if !blacklist.IsAddressBlacklisted(&log.Address, epochNumber) {
					logs = append(logs, log)
				}
				txLogIndex++
				logIndex++
			}
			receipt.Logs = logs
			receipts[tx.Hash] = receipt
		}
	}

	return EpochData{
		Number: epochNumber, Blocks: blocks, Receipts: receipts,
	}, nil
}

func validateBlock(block *types.Block, epochNumber uint64, hash types.Hash) error {
	if block.EpochNumber == nil {
		return errors.WithMessage(errBlockValidationFailed, "epoch number is nil")
	}

	bEpochNumber := block.EpochNumber.ToInt().Uint64()
	if bEpochNumber != epochNumber {
		errMsg := fmt.Sprintf(
			"epoch number mismatched, expect %v got %v", epochNumber, bEpochNumber,
		)
		return errors.WithMessage(errBlockValidationFailed, errMsg)
	}

	if block.Hash != hash {
		return errors.WithMessage(errBlockValidationFailed, "block hash not matched")
	}

	return nil
}

func validateTxsReceipt(
	receipt *types.TransactionReceipt, epochNumber uint64,
	block *types.Block, tx *types.Transaction,
) error {
	// Check receipt epoch number
	rcptEpochNumber := uint64(*receipt.EpochNumber)
	if rcptEpochNumber != epochNumber {
		errMsg := fmt.Sprintf(
			"epoch number mismatched, expect %v got %v", epochNumber, rcptEpochNumber,
		)
		return errors.WithMessage(errTxsReceiptValidationFailed, errMsg)
	}

	// Check receipt block hash
	if receipt.BlockHash != block.Hash {
		errMsg := fmt.Sprintf(
			"block hash mismatched, expect %v got %v", block.Hash, receipt.BlockHash,
		)
		return errors.WithMessage(errTxsReceiptValidationFailed, errMsg)
	}

	// Check receipt transaction hash
	if receipt.TransactionHash != tx.Hash {
		errMsg := fmt.Sprintf(
			"txs hash mismatched, expect %v got %v", tx.Hash, receipt.TransactionHash,
		)
		return errors.WithMessage(errTxsReceiptValidationFailed, errMsg)
	}

	return nil
}

// Check if epoch pivot switched from query or validation error.
func checkPivotSwitchWithError(err error) bool {
	if err == nil {
		return false
	}

	// The error is an epoch block validation error, take it as pivot switched.
	if errors.Is(err, errBlockValidationFailed) {
		return true
	}

	// The error is detected as a business error and pivot hash assumption failed, must be pivot switched.
	detected, errCode := sdkerr.DetectErrorCode(err)
	if detected && (errCode == sdkerr.CodePivotAssumption || errCode == sdkerr.CodeBlockNotFound) {
		return true
	}

	return false
}
