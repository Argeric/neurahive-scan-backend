package store

import (
	"encoding/hex"
	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"github.com/Conflux-Chain/neurahive-client/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go/types"
	"gorm.io/gorm"
	"time"
)

type Submit struct {
	ID               uint64     `gorm:"primaryKey;index:idx_sender_id,priority:2"`
	BlockNumber      uint64     `gorm:"not null;index:idx_bn"`
	TxHash           string     `gorm:"type:varchar(64);not null;index:idx_hash,length:10"`
	CreatedAt        *time.Time `gorm:"not null;index:idx_createdAt,sort:desc"`
	Sender           string     `gorm:"-"`
	SenderId         uint64     `gorm:"not null;index:idx_sender_id,priority:1"`
	Identity         string     `gorm:"size:64;not null"`
	SubmissionIndex  uint64     `gorm:"not null"`
	StartPos         uint64     `gorm:"not null"`
	Length           uint64     `gorm:"not null"`
	SubmissionLength uint64     `gorm:"not null"`
}

func NewSubmit(blockTime *time.Time, log *types.Log) (*Submit, error) {
	contract, _ := contract.NewFlowFilterer(common.HexToAddress(""), nil)
	flowSubmit, err := contract.ParseSubmit(*log.ToEthLog())
	if err != nil {
		return nil, err
	}

	submit := &Submit{
		BlockNumber:      log.BlockNumber,
		TxHash:           log.TxHash.String()[2:],
		CreatedAt:        blockTime,
		Sender:           flowSubmit.Sender.String()[2:],
		Identity:         hex.EncodeToString(flowSubmit.Identity[:]),
		SubmissionIndex:  flowSubmit.SubmissionIndex.Uint64(),
		StartPos:         flowSubmit.StartPos.Uint64(),
		Length:           flowSubmit.Length.Uint64(),
		SubmissionLength: flowSubmit.Submission.Length.Uint64(),
	}

	return submit, nil
}

func (Submit) TableName() string {
	return "submits"
}

type SubmitStore struct {
	*mysql.Store
}

func newSubmitStore(db *gorm.DB) *SubmitStore {
	return &SubmitStore{
		Store: mysql.NewStore(db),
	}
}

func (ss *SubmitStore) Add(dbTx *gorm.DB, submits []*Submit) error {
	return dbTx.CreateInBatches(submits, batchSizeInsert).Error
}

func (ss *SubmitStore) Pop(dbTx *gorm.DB, block uint64) error {
	return dbTx.Where("block_number >= ?", block).Delete(&Submit{}).Error
}

func (ss *SubmitStore) Count(startTime, endTime *time.Time) (uint64, uint64, error) {
	var result struct {
		FileCount int64
		DataSize  int64
	}
	err := ss.DB.Model(&Submit{}).Select("count(id) as file_count, IFNULL(sum(submission_length), 0) as data_size").
		Where("created_at >= ? and created_at < ?", startTime, endTime).Find(&result).Error
	if err != nil {
		return 0, 0, err
	}

	return uint64(result.FileCount), uint64(result.DataSize), nil
}
