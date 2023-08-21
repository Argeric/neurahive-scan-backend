package service

import (
	"context"
	"github.com/Argeric/neurahive-scan-backend/store"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"sync"
)

type syncConfig struct {
}

type EpochSyncer struct {
	conf *syncConfig
	cfx  sdk.ClientOperator
	db   *store.MysqlStore
}

// MustNewDatabaseSyncer creates an instance of DatabaseSyncer to sync blockchain data.
func MustNewDatabaseSyncer(cfx sdk.ClientOperator, db *store.MysqlStore) *EpochSyncer {
	var conf syncConfig
	viperutil.MustUnmarshalKey("sync", &conf)

	syncer := &EpochSyncer{
		conf: &conf,
		cfx:  cfx,
		db:   db,
	}

	return syncer
}

func (es *EpochSyncer) Sync(ctx context.Context, wg *sync.WaitGroup) {

}
