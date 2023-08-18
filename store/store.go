package store

import (
	"github.com/Argeric/neurahive-scan-backend/config"
	"gorm.io/gorm"
)

type MysqlStore struct {
	*baseStore
	*epochStore
	*blockStore
	*txStore
	*submitStore

	config *config.DBConfig
}

func MustNewStore(db *gorm.DB, config *config.DBConfig) *MysqlStore {
	return &MysqlStore{
		baseStore:   newBaseStore(db),
		epochStore:  newEpochStore(db),
		blockStore:  newBlockStore(db),
		txStore:     newTxStore(db),
		submitStore: newSubmitStore(db),

		config: config,
	}
}
