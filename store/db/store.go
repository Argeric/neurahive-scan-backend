package db

import (
	"github.com/Argeric/neurahive-scan-backend/store/blockchain"
	"gorm.io/gorm"
)

type MysqlStore struct {
	*baseStore
	*epochStore
	*blockStore
	*txStore
	*submitStore

	config *Config
}

func MustNewStore(db *gorm.DB, config *Config) *MysqlStore {
	return &MysqlStore{
		baseStore:   newBaseStore(db),
		epochStore:  newEpochStore(db),
		blockStore:  newBlockStore(db),
		txStore:     newTxStore(db),
		submitStore: newSubmitStore(db),

		config: config,
	}
}

func (ms *MysqlStore) Push(data *blockchain.EpochData) error {
	return nil
}

func (ms *MysqlStore) Pop(epoch uint64) error {
	return nil
}
