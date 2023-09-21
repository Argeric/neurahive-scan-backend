package cmd

import (
	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/Conflux-Chain/neurahive-scan/store"
	"github.com/openweb3/web3go"
	"github.com/sirupsen/logrus"
	"time"
)

// SyncContext context to hold sdk clients for blockchain interoperation.
type SyncContext struct {
	Eth *web3go.Client
	DB  *store.MysqlStore
}

type SdkConfig struct {
	Url             string
	Retry           int
	RetryInterval   time.Duration `default:"1s"`
	RequestTimeout  time.Duration `default:"3s"`
	MaxConnsPerHost int           `default:"1024"`
}

var migrationModels = []interface{}{
	&store.Address{},
	&store.Block{},
	&store.Submit{},
	&store.Tx{},
}

func MustInitSyncContext() SyncContext {
	cfg := mysql.MustNewConfigFromViper()
	db := cfg.MustOpenOrCreate()
	if err := db.AutoMigrate(migrationModels...); err != nil {
		logrus.WithError(err).Fatalln("failed to migrate database")
	}

	sdkCfg := SdkConfig{}
	viper.MustUnmarshalKey("eth", &sdkCfg)
	opt := web3go.ClientOption{}
	opt.WithRetry(sdkCfg.Retry, sdkCfg.RetryInterval).
		WithTimout(sdkCfg.RequestTimeout).
		WithMaxConnectionPerHost(sdkCfg.MaxConnsPerHost)
	eth := web3go.MustNewClientWithOption(sdkCfg.Url, opt)

	return SyncContext{
		DB:  store.MustNewStore(db),
		Eth: eth,
	}
}

func (ctx *SyncContext) Close() {
	if ctx.DB != nil {
		ctx.DB.Close()
	}

	if ctx.Eth != nil {
		ctx.Eth.Close()
	}
}
