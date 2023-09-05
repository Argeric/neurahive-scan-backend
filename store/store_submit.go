package store

import (
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/Conflux-Chain/neurahive-client/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go/types"
	"gorm.io/gorm"
	"strings"
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

func newSubmit(blockTime *time.Time, log *types.Log) (*Submit, error) {
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
		Identity:         string(flowSubmit.Identity[:]),
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

type submitStore struct {
	as            *addressStore
	flowAddr      string
	flowSubmitSig string
}

func newSubmitStore(db *gorm.DB) *submitStore {
	var flow struct {
		Address              string
		SubmitEventSignature string
	}
	viperutil.MustUnmarshalKey("flow", &flow)

	return &submitStore{
		as:            newAddressStore(db),
		flowAddr:      flow.Address,
		flowSubmitSig: flow.SubmitEventSignature,
	}
}

func (ss *submitStore) Add(dbTx *gorm.DB, data *EthData) error {
	block := data.Block
	blockTime := time.Unix(int64(block.Timestamp), 0)

	var submits []*Submit
	for _, tx := range block.Transactions.Transactions() {
		receipt := data.Receipts[tx.Hash]
		if receipt == nil || !IsTxExecutedInBlock(&tx, receipt) {
			continue
		}

		for _, log := range receipt.Logs {
			contract := log.Address.String()
			topic0 := log.Topics[0].String()
			if !strings.EqualFold(contract, ss.flowAddr) || topic0 != ss.flowSubmitSig {
				continue
			}

			submit, err := newSubmit(&blockTime, log)
			if err != nil {
				return err
			}

			senderId, err := ss.as.Add(nil, submit.Sender, &blockTime)
			if err != nil {
				return err
			}

			submit.SenderId = senderId
			submits = append(submits, submit)
		}
	}

	if len(submits) == 0 {
		return nil
	}

	return dbTx.CreateInBatches(submits, batchSizeInsert).Error
}

func (ss *submitStore) Pop(dbTx *gorm.DB, block uint64) error {
	return dbTx.Where("block_number >= ?", block).Delete(&Submit{}).Error
}
