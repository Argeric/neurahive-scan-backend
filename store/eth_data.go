package store

import (
	"context"
	set "github.com/deckarep/golang-set"
	"github.com/ethereum/go-ethereum/common"
	rpc "github.com/openweb3/go-rpc-provider"
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
	blockTxHashes := block.Transactions.Hashes()
	if len(blockTxHashes) != len(blockReceipts) {
		return nil, errors.Errorf("block receipts number mismatch, rcpts %v, txs %v", len(blockReceipts), len(blockTxHashes))
	}
	for i := 0; i < len(blockTxHashes); i++ {
		txHash := blockTxHashes[i]
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

func QueryFlowSubmits(w3c *web3go.Client, blockFrom, blockTo uint64, flowAddr common.Address, flowSubmitSig common.Hash) ([]types.Log, error) {
	bnFrom := types.NewBlockNumber(int64(blockFrom))
	bnTo := types.NewBlockNumber(int64(blockTo))
	logFilter := types.FilterQuery{
		FromBlock: &bnFrom,
		ToBlock:   &bnTo,
		Addresses: []common.Address{flowAddr},
		Topics:    [][]common.Hash{{flowSubmitSig}},
	}
	return w3c.Eth.Logs(logFilter)
}

func MapBlockNum2Time(ctx context.Context, w3c *web3go.Client, blkNums []types.BlockNumber, batchSize uint64) (map[uint64]uint64, error) {
	if len(blkNums) == 0 {
		return nil, errors.New("no block numbers")
	}

	blkNumSet := set.NewSet()
	for _, num := range blkNums {
		blkNumSet.Add(num)
	}

	blockNum2Time := make(map[uint64]uint64)
	blkNumSlice := blkNumSet.ToSlice()
	blkNumSize := len(blkNumSlice)
	for i := 0; i < blkNumSize; i += int(batchSize) {
		end := i + int(batchSize)
		if end > blkNumSize {
			end = blkNumSize
		}
		blockNums := blkNumSlice[i:end]

		batch := make([]rpc.BatchElem, 0)
		for _, blkNum := range blockNums {
			elem := rpc.BatchElem{
				Method: "eth_getBlockByNumber",
				Args:   []interface{}{blkNum, false},
				Result: new(types.Block),
			}
			batch = append(batch, elem)
		}

		err := w3c.Eth.BatchCallContext(ctx, batch)
		if err != nil {
			return nil, err
		}

		for _, elem := range batch {
			block := elem.Result.(*types.Block)
			blockNum2Time[block.Number.Uint64()] = block.Timestamp
		}
	}

	return blockNum2Time, nil
}
