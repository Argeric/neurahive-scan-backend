package store

import (
	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	batchSizeInsert = 100
)

var (
	ErrNotFound     = errors.New("not found")
	ErrChainReorged = errors.New("chain re-orged")
)

type MysqlStore struct {
	baseStore *mysql.Store
	*blockStore
	*submitStore
	*addressStore
}

func MustNewStore(db *gorm.DB) *MysqlStore {
	return &MysqlStore{
		baseStore:    mysql.NewStore(db),
		blockStore:   newBlockStore(db),
		submitStore:  newSubmitStore(db),
		addressStore: newAddressStore(db),
	}
}

func (ms *MysqlStore) Push(data *EthData) error {
	return ms.baseStore.DB.Transaction(func(dbTx *gorm.DB) error {
		// save blocks
		if err := ms.blockStore.Add(dbTx, data); err != nil {
			return errors.WithMessagef(err, "failed to save block")
		}

		// save flow submits
		if err := ms.submitStore.Add(dbTx, data); err != nil {
			return errors.WithMessage(err, "failed to save flow submits")
		}

		return nil
	})
}

func (ms *MysqlStore) Pop(block uint64) error {
	maxBlock, ok, err := ms.MaxBlock()
	if err != nil {
		return errors.WithMessage(err, "failed to get max block")
	}
	if !ok || block > maxBlock {
		return nil
	}

	return ms.baseStore.DB.Transaction(func(dbTx *gorm.DB) error {
		if err := ms.blockStore.Pop(dbTx, block); err != nil {
			return errors.WithMessage(err, "failed to remove block")
		}
		if err := ms.submitStore.Pop(dbTx, block); err != nil {
			return errors.WithMessage(err, "failed to remove flow submits")
		}
		return nil
	})
}
