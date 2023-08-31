package store

import (
	"github.com/Conflux-Chain/neurahive-client/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/openweb3/web3go/types"
	"gorm.io/gorm"
	"time"
)

type Submit struct {
	BlockNumber      uint64     `gorm:"primary_key;autoIncrement:false"`
	TxPosition       uint16     `gorm:"primary_key;autoIncrement:false"`
	TxLogPosition    uint16     `gorm:"primary_key;autoIncrement:false"`
	Contract         string     `gorm:"-"`
	ContractId       uint64     `gorm:"not null"`
	CreatedAt        *time.Time `gorm:"not null;index:idx_createdAt,sort:desc"`
	Sender           string     `gorm:"-"`
	SenderId         uint64     `gorm:"not null;index:idx_sender"`
	Identity         string     `gorm:"size:64;not null"`
	SubmissionIndex  uint64     `gorm:"not null"`
	StartPos         uint64     `gorm:"not null"`
	Length           uint64     `gorm:"not null"`
	SubmissionLength uint64     `gorm:"not null"`
}

func newSubmit(blockTime *time.Time, log *types.Log, txIndex, txLogIndex int) (*Submit, error) {
	contract, _ := contract.NewFlowFilterer(common.HexToAddress(""), nil)
	flowSubmit, err := contract.ParseSubmit(*log.ToEthLog())
	if err != nil {
		return nil, err
	}

	submit := &Submit{
		BlockNumber:      log.BlockNumber,
		TxPosition:       uint16(txIndex),
		TxLogPosition:    uint16(txLogIndex),
		Contract:         log.Address.String()[2:],
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

type AddressSubmit struct {
	AddressId        uint64     `gorm:"primary_key;autoIncrement:false"`
	BlockNumber      uint64     `gorm:"primary_key;autoIncrement:false"`
	TxPosition       uint16     `gorm:"primary_key;autoIncrement:false"`
	TxLogPosition    uint16     `gorm:"primary_key;autoIncrement:false"`
	ContractId       uint64     `gorm:"not null"`
	CreatedAt        *time.Time `gorm:"not null;index:idx_createdAt,sort:desc"`
	Identity         string     `gorm:"size:64;not null"`
	SubmissionIndex  uint64     `gorm:"not null"`
	StartPos         uint64     `gorm:"not null"`
	Length           uint64     `gorm:"not null"`
	SubmissionLength uint64     `gorm:"not null"`
}

func newAddressSubmit(submit *Submit) *AddressSubmit {
	return &AddressSubmit{
		AddressId:        submit.SenderId,
		BlockNumber:      submit.BlockNumber,
		TxPosition:       submit.TxPosition,
		TxLogPosition:    submit.TxLogPosition,
		ContractId:       submit.ContractId,
		CreatedAt:        submit.CreatedAt,
		Identity:         submit.Identity,
		SubmissionIndex:  submit.SubmissionIndex,
		StartPos:         submit.StartPos,
		Length:           submit.Length,
		SubmissionLength: submit.SubmissionLength,
	}
}

func (AddressSubmit) TableName() string {
	return "address_submits"
}

type submitStore struct {
	as *addressStore
}

func newSubmitStore(db *gorm.DB) *submitStore {
	return &submitStore{
		as: newAddressStore(db),
	}
}

func (ss *submitStore) Add(dbTx *gorm.DB, data *EthData) error {
	var submits []*Submit
	var addressSubmits []*AddressSubmit

	block := data.Block
	blockTime := time.Unix(int64(block.Timestamp), 0)

	for j, tx := range block.Transactions.Transactions() {
		receipt := data.Receipts[tx.Hash]
		if receipt == nil || !IsTxExecutedInBlock(&tx, receipt) {
			continue
		}
		for k, log := range receipt.Logs {
			submit, err := newSubmit(&blockTime, log, j, k)
			if err != nil {
				return err
			}

			contractId, err := ss.as.Add(nil, submit.Contract, &blockTime)
			if err != nil {
				return err
			}
			senderId, err := ss.as.Add(nil, submit.Sender, &blockTime)
			if err != nil {
				return err
			}

			submit.ContractId = contractId
			submit.SenderId = senderId
			submits = append(submits, submit)
			addressSubmits = append(addressSubmits, newAddressSubmit(submit))
		}
	}

	if len(submits) == 0 {
		return nil
	}

	if err := dbTx.CreateInBatches(submits, batchSizeInsert).Error; err != nil {
		return err
	}
	return dbTx.CreateInBatches(addressSubmits, batchSizeInsert).Error
}
