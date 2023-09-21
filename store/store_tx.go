package store

import (
	"github.com/openweb3/web3go/types"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"time"
)

type Tx struct {
	ID          uint64           `gorm:"primaryKey"`
	BlockNumber uint64           `gorm:"not null;index:idx_bn"`
	Hash        string           `gorm:"type:varchar(64);not null;index:idx_hash,length:10"`
	From        string           `gorm:"-"`
	FromId      uint64           `gorm:"not null"`
	To          string           `gorm:"-"`
	ToId        uint64           `gorm:"not null"`
	Nonce       uint64           `gorm:"not null"`
	MethodId    string           `gorm:"type:varchar(8);default:null"` // MethodId is function selector
	DripValue   *decimal.Decimal `gorm:"type:varchar(78);not null"`
	GasPrice    uint64           `gorm:"not null;default:0"`
	Gas         uint64           `gorm:"not null;default:0"`
	Status      uint64           `gorm:"not null;default:0"`
	CreatedAt   *time.Time       `gorm:"not null;index:idx_createdAt,sort:desc"`
}

func NewTx(blockTime *time.Time, tx *types.TransactionDetail) *Tx {
	val := decimal.NewFromBigInt(tx.Value, 0)
	return &Tx{
		BlockNumber: tx.BlockNumber.Uint64(),
		Hash:        tx.Hash.String()[2:],
		From:        tx.From.String()[2:],
		To:          tx.To.String()[2:],
		Nonce:       tx.Nonce,
		MethodId:    tx.Input.String()[2:10],
		DripValue:   &val,
		GasPrice:    tx.GasPrice.Uint64(),
		Gas:         tx.Gas,
		Status:      *tx.Status,
		CreatedAt:   blockTime,
	}
}

func (Tx) TableName() string {
	return "txs"
}

type txStore struct {
	as *AddressStore
}

func newTxStore(db *gorm.DB) *txStore {
	return &txStore{
		as: newAddressStore(db),
	}
}

func (ts *txStore) Add(dbTx *gorm.DB, txs []*Tx) error {
	return dbTx.CreateInBatches(txs, batchSizeInsert).Error
}

func (ts *txStore) Pop(dbTx *gorm.DB, block uint64) error {
	return dbTx.Where("block_number >= ?", block).Delete(&Tx{}).Error
}
