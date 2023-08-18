package store

import (
	"gorm.io/gorm"
	"time"
)

type Epoch struct {
	Epoch     uint64     `gorm:"primary_key;autoIncrement:false"`
	PivotHash string     `gorm:"type:varchar(66);not null;index:idx_hash,length:10"`
	timestamp *time.Time `gorm:"not null;index:idx_timestamp,sort:desc"`
}

func (Epoch) TableName() string {
	return "epoch"
}

type epochStore struct {
	*baseStore
}

func newEpochStore(db *gorm.DB) *epochStore {
	return &epochStore{
		baseStore: newBaseStore(db),
	}
}
