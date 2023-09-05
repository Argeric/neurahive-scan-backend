package store

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go"
	"github.com/openweb3/web3go/types"
	"github.com/pkg/errors"
)

type EthData struct {
	Number   uint64
	Block    *types.Block
	Receipts map[common.Hash]*types.Receipt
}

func IsTxExecutedInBlock(tx *types.TransactionDetail, receipt *types.Receipt) bool {
	return tx != nil && receipt.Status != nil && *receipt.Status < 2
}

func QueryEthData(w3c *web3go.Client, blockNumber uint64) (*EthData, error) {
	// get block
	block, err := w3c.Eth.BlockByNumber(types.BlockNumber(blockNumber), false)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get block by number %v", blockNumber)
	}
	if block == nil {
		return nil, nil
	}

	// batch get receipts
	blockNumOrHash := types.BlockNumberOrHashWithNumber(types.BlockNumber(blockNumber))
	blockReceipts, err := w3c.Parity.BlockReceipts(&blockNumOrHash)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get block receipts")
	}

	// get receipt
	txReceipts := map[common.Hash]*types.Receipt{}
	blockTxs := block.Transactions.Transactions()
	if len(blockTxs) != len(blockReceipts) {
		return nil, errors.Errorf("block receipts number mismatch, rcpts %v, txs %v", len(blockReceipts), len(blockTxs))
	}
	for i := 0; i < len(blockTxs); i++ {
		txHash := blockTxs[i].Hash
		var receipt *types.Receipt
		if blockReceipts == nil {
			return nil, errors.WithMessage(ErrChainReorged, "batch retrieved block receipts nil")
		}
		receipt = &blockReceipts[i]

		// check re-org
		switch {
		case receipt == nil: // receipt shouldn't be nil unless chain re-org
			return nil, errors.WithMessage(ErrChainReorged, "tx receipt nil")
		case receipt.BlockHash != block.Hash:
			return nil, errors.WithMessagef(ErrChainReorged, "receipt block hash mismatch, rcptBlkHash %v, blkHash %v",
				receipt.BlockHash, block.Hash)
		case receipt.BlockNumber != blockNumber:
			return nil, errors.WithMessagef(ErrChainReorged, "receipt block num mismatch, rcptBlkNum %v, blkNum %v",
				receipt.BlockNumber, blockNumber)
		case receipt.TransactionHash != txHash:
			return nil, errors.WithMessagef(ErrChainReorged, "receipt tx hash mismatch, rcptTxHash %v, TxHash %v",
				receipt.TransactionHash, txHash)
		}
		txReceipts[txHash] = receipt
	}

	return &EthData{blockNumber, block, txReceipts}, nil
}
