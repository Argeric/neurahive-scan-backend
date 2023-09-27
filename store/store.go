package store

import (
	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	batchSizeInsert = 100
)

type MysqlStore struct {
	*mysql.Store
	*AddressStore
	*BlockStore
	*submitStore
	*TxStore
	*TxStatStore
}

func MustNewStore(db *gorm.DB) *MysqlStore {
	return &MysqlStore{
		Store:        mysql.NewStore(db),
		AddressStore: newAddressStore(db),
		BlockStore:   newBlockStore(db),
		submitStore:  newSubmitStore(db),
		TxStore:      newTxStore(db),
		TxStatStore:  newTxStatStore(db),
	}
}

func (ms *MysqlStore) Push(block *Block, txs []*Tx, submits []*Submit) error {

	return ms.Store.DB.Transaction(func(dbTx *gorm.DB) error {
		// save blocks
		if err := ms.BlockStore.Add(dbTx, block); err != nil {
			return errors.WithMessage(err, "failed to save block")
		}

		// save txs
		if len(txs) > 0 {
			if err := ms.TxStore.Add(dbTx, txs); err != nil {
				return errors.WithMessage(err, "failed to save txs")
			}
		}

		// save flow submits
		if len(submits) > 0 {
			if err := ms.submitStore.Add(dbTx, submits); err != nil {
				return errors.WithMessage(err, "failed to save flow submits")
			}
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

	return ms.Store.DB.Transaction(func(dbTx *gorm.DB) error {
		if err := ms.BlockStore.Pop(dbTx, block); err != nil {
			return errors.WithMessage(err, "failed to remove block")
		}
		if err := ms.TxStore.Pop(dbTx, block); err != nil {
			return errors.WithMessage(err, "failed to remove txs")
		}
		if err := ms.submitStore.Pop(dbTx, block); err != nil {
			return errors.WithMessage(err, "failed to remove flow submits")
		}
		return nil
	})
}

func (ms *MysqlStore) Close() error {
	return ms.Store.Close()
}
