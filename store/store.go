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
	*txStore
	*submitStore
	*addressStore
}

func MustNewStore(db *gorm.DB) *MysqlStore {
	return &MysqlStore{
		baseStore:    mysql.NewStore(db),
		blockStore:   newBlockStore(db),
		txStore:      newTxStore(db),
		submitStore:  newSubmitStore(db),
		addressStore: newAddressStore(db),
	}
}

func (ms *MysqlStore) Push(data *EthData) error {
	return ms.baseStore.DB.Transaction(func(dbTx *gorm.DB) error {
		// save blocks
		if err := ms.blockStore.Add(dbTx, data); err != nil {
			return errors.WithMessagef(err, "failed to save blocks")
		}

		// save transactions
		if err := ms.txStore.Add(dbTx, data); err != nil {
			return errors.WithMessage(err, "failed to save txs")
		}

		// save submit event logs
		if err := ms.submitStore.Add(dbTx, data); err != nil {
			return errors.WithMessage(err, "failed to save submit event logs")
		}

		return nil
	})
}

func (ms *MysqlStore) Pop(block uint64) error {
	// TODO
	return nil
}
