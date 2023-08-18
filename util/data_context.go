package util

import (
	"github.com/Argeric/neurahive-scan-backend/config"
	"github.com/Argeric/neurahive-scan-backend/store"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
)

// SyncContext context to hold sdk clients for blockchain interoperation.
type SyncContext struct {
	DB  *store.MysqlStore
	Cfx *sdk.Client
}

func MustInitSyncContext() SyncContext {
	var ctx SyncContext

	if config := config.MustNewConfigFromViper("database"); config.Enabled {
		ctx.DB = config.MustOpenOrCreate()
	}

	if storeCtx.CfxDB != nil || storeCtx.CfxCache != nil {
		ctx.Cfx = rpc.MustNewCfxClientFromViper()
	}

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
