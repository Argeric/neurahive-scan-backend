package util

import (
	"github.com/Argeric/neurahive-scan-backend/config"
	"github.com/Argeric/neurahive-scan-backend/store"
	"github.com/Argeric/neurahive-scan-backend/util/rpc"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
)

// SyncContext context to hold sdk clients for blockchain interoperation.
type SyncContext struct {
	Cfx *sdk.Client
	DB  *store.MysqlStore
}

func MustInitSyncContext() SyncContext {
	var ctx SyncContext

	if config := config.MustNewConfigFromViper("database"); config.Enabled {
		ctx.DB = config.MustOpenOrCreate()
	}

	ctx.Cfx = rpc.MustNewCfxClientFromViper()

	return ctx
}

func (ctx *SyncContext) Close() {
	if ctx.DB != nil {
		ctx.DB.Close()
	}

	if ctx.Cfx != nil {
		ctx.Cfx.Close()
	}
}
