package db

import (
	"gorm.io/gorm"
	"time"
)

type Submit struct {
	Epoch         uint64     `gorm:"primary_key;autoIncrement:false"`
	BlockPosition uint64     `gorm:"primary_key;autoIncrement:false"`
	TxPosition    uint64     `gorm:"primary_key;autoIncrement:false"`
	TxLogPosition uint64     `gorm:"primary_key;autoIncrement:false"`
	ContractId    uint64     `gorm:"not null"`
	CreatedAt     *time.Time `gorm:"not null;index:idx_createdAt,sort:desc"`

	SenderId        uint64 `gorm:"not null"`
	Identity        string `gorm:"not null"`
	SubmissionIndex uint   `gorm:"not null"`
	StartPos        uint   `gorm:"not null"`
	Length          uint   `gorm:"not null"`

	SubmissionLength uint   `gorm:"not null"`
	SubmissionTags   string `gorm:"not null"`
	SubmissionNodes  string `gorm:"not null"`
}

func (Submit) TableName() string {
	return "submit"
}

type submitStore struct {
	*baseStore
}

func newSubmitStore(db *gorm.DB) *submitStore {
	return &submitStore{
		baseStore: newBaseStore(db),
	}
}
