package db

import (
	"gorm.io/gorm"
	"time"
)

type Block struct {
	Epoch            uint64     `gorm:"primary_key;autoIncrement:false"`
	Position         uint64     `gorm:"primary_key;autoIncrement:false;default:0"`
	CreatedAt        *time.Time `gorm:"not null;index:idx_block_time,sort:desc"`
	Difficulty       uint64     `gorm:"not null;default:0"`
	MinerId          uint64     `gorm:"not null"`
	Hash             string     `gorm:"type:varchar(66);not null;index:idx_hash,length:10"`
	TotalReward      uint64     `gorm:"not null;default:0"`
	TxFee            uint64     `gorm:"not null;default:0"`
	AvgGasPrice      uint64     `gorm:"not null;default:0"`
	GasLimit         uint64     `gorm:"not null;default:0"`
	GasUsed          uint64     `gorm:"not null;default:0"`
	TxCount          uint64     `gorm:"not null;default:0"` // all txn, include packed but not executed
	ExecutedTxnCount uint64     `gorm:"not null;default:0"`
	Pivot            bool       `gorm:"not null;default:false"`
}

func (Block) TableName() string {
	return "block"
}

type blockStore struct {
	*baseStore
}

func newBlockStore(db *gorm.DB) *blockStore {
	return &blockStore{
		baseStore: newBaseStore(db),
	}
}
