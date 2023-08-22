package store

import (
	"database/sql"
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

// MaxEpoch returns the max epoch within the map store.
func (es *epochStore) MaxEpoch() (uint64, bool, error) {
	var maxEpoch sql.NullInt64

	db := es.db.Model(&Epoch{}).Select("MAX(epoch)")
	if err := db.Find(&maxEpoch).Error; err != nil {
		return 0, false, err
	}

	if !maxEpoch.Valid {
		return 0, false, nil
	}

	return uint64(maxEpoch.Int64), true, nil
}

// pivotHash returns the pivot hash of the given epoch.
func (es *epochStore) PivotHash(epoch uint64) (string, bool, error) {
	var ep Epoch

	existed, err := es.Exists(&ep, "epoch = ?", epoch)
	if err != nil {
		return "", false, err
	}

	return ep.PivotHash, existed, nil
}
