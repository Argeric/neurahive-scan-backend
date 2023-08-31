package store

import (
	"github.com/openweb3/web3go/types"
	"gorm.io/gorm"
	"time"
)

type Tx struct {
	BlockNumber uint64     `gorm:"primary_key;autoIncrement:false"`
	TxPosition  uint64     `gorm:"primary_key;autoIncrement:false"`
	Hash        string     `gorm:"type:varchar(64);not null;index:idx_hash,length:10"`
	From        string     `gorm:"-"`
	FromId      uint64     `gorm:"not null"`
	Nonce       uint64     `gorm:"not null"`
	To          string     `gorm:"-"`
	ToId        uint64     `gorm:"not null"`
	DripValue   uint64     `gorm:"not null;default:0"`
	GasPrice    uint64     `gorm:"not null;default:0"`
	Gas         uint64     `gorm:"not null;default:0"`
	Status      uint64     `gorm:"not null"`
	InterfaceId string     `gorm:"type:varchar(10);not null"` // interfaceId is function selector
	CreatedAt   *time.Time `gorm:"not null;index:idx_createdAt,sort:desc"`
}

func newTx(blockTime *time.Time, tx *types.TransactionDetail, receipt *types.Receipt, txIndex int) *Tx {
	return &Tx{
		BlockNumber: receipt.BlockNumber,
		TxPosition:  uint64(txIndex),
		Hash:        receipt.TransactionHash.String()[2:],
		From:        tx.From.String()[2:],
		Nonce:       tx.Nonce,
		To:          tx.To.String()[2:],
		DripValue:   tx.Value.Uint64(),
		GasPrice:    tx.GasPrice.Uint64(),
		Gas:         tx.Gas,
		Status:      *receipt.Status,
		CreatedAt:   blockTime,
	}
}

func (Tx) TableName() string {
	return "txs"
}

type AddressTx struct {
	AddressId   uint64     `gorm:"primary_key;autoIncrement:false"`
	BlockNumber uint64     `gorm:"primary_key;autoIncrement:false"`
	TxPosition  uint64     `gorm:"primary_key;autoIncrement:false"`
	Hash        string     `gorm:"type:varchar(64);not null;index:idx_hash,length:10"`
	FromId      uint64     `gorm:"not null"`
	Nonce       uint64     `gorm:"not null"`
	ToId        uint64     `gorm:"not null"`
	DripValue   uint64     `gorm:"not null;default:0"`
	GasPrice    uint64     `gorm:"not null;default:0"`
	Gas         uint64     `gorm:"not null;default:0"`
	Status      uint64     `gorm:"not null"`
	CreatedAt   *time.Time `gorm:"not null;index:idx_createdAt,sort:desc"`
}

func newAddressTx(tx *Tx) []*AddressTx {
	addrIds := []uint64{tx.FromId}
	if tx.ToId != tx.FromId {
		addrIds = append(addrIds, tx.ToId)
	}

	var addrTx []*AddressTx
	for _, addrId := range addrIds {
		addrTx = append(addrTx, &AddressTx{
			AddressId:   addrId,
			BlockNumber: tx.BlockNumber,
			TxPosition:  tx.TxPosition,
			FromId:      tx.FromId,
			Hash:        tx.Hash,
			ToId:        tx.ToId,
			Nonce:       tx.Nonce,
			DripValue:   tx.DripValue,
			GasPrice:    tx.GasPrice,
			Gas:         tx.Gas,
			Status:      tx.Status,
			CreatedAt:   tx.CreatedAt,
		})
	}

	return addrTx
}

func (AddressTx) TableName() string {
	return "address_txs"
}

type txStore struct {
	as *addressStore
}

func newTxStore(db *gorm.DB) *txStore {
	return &txStore{
		as: newAddressStore(db),
	}
}

func (ts *txStore) Add(dbTx *gorm.DB, data *EthData) error {
	var txs []*Tx
	var addressTxs []*AddressTx

	block := data.Block
	blockTime := time.Unix(int64(block.Timestamp), 0)

	for i, tx := range block.Transactions.Transactions() {
		receipt := data.Receipts[tx.Hash]
		if receipt == nil || !IsTxExecutedInBlock(&tx, receipt) {
			continue
		}

		tx := newTx(&blockTime, &tx, receipt, i)
		fromId, err := ts.as.Add(nil, tx.From, &blockTime)
		if err != nil {
			return err
		}
		toId, err := ts.as.Add(nil, tx.To, &blockTime)
		if err != nil {
			return err
		}
		tx.FromId = fromId
		tx.ToId = toId

		txs = append(txs, tx)
		addressTxs = append(addressTxs, newAddressTx(tx)...)
	}

	if len(txs) == 0 {
		return nil
	}

	if err := dbTx.CreateInBatches(txs, batchSizeInsert).Error; err != nil {
		return err
	}
	return dbTx.CreateInBatches(addressTxs, batchSizeInsert).Error

}
