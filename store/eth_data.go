package store

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go/types"
)

type EthData struct {
	Number   uint64
	Block    *types.Block
	Receipts map[common.Hash]*types.Receipt
}

func IsTxExecutedInBlock(tx *types.TransactionDetail, receipt *types.Receipt) bool {
	return tx != nil && receipt.Status != nil && *receipt.Status < 2
}
