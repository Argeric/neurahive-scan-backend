package cmd

import (
	"github.com/Argeric/neurahive-scan-backend/store/db"
	"github.com/Argeric/neurahive-scan-backend/util/rpc"
	sdk "github.com/Conflux-Chain/go-conflux-sdk"
)

// SyncContext context to hold sdk clients for blockchain interoperation.
type SyncContext struct {
	Cfx *sdk.Client
	DB  *db.MysqlStore
}

func MustInitSyncContext() SyncContext {
	var ctx SyncContext

	if config := db.MustNewConfigFromViper("database"); config.Enabled {
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
