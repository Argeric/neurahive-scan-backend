package api

import (
	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/Conflux-Chain/neurahive-scan/store"
	"github.com/gin-gonic/gin"
	"github.com/openweb3/web3go"
)

var (
	sdk *web3go.Client
	db  *store.MysqlStore
)

func Init(client *web3go.Client, store *store.MysqlStore) {
	sdk = client
	db = store
}

func RegisterRouter(router *gin.Engine) {
	apiRoute := router.Group("/api")

	statRoute := apiRoute.Group("/statistic")
	statRoute.GET("transaction/list", api.Wrap(listTxStat))
	statRoute.GET("storage/list", api.Wrap(listDataStat))

	txRoute := apiRoute.Group("/transaction")
	txRoute.GET("list", api.Wrap(listTx))
	txRoute.GET("brief", api.Wrap(listTxBrief))
	txRoute.GET("detail", api.Wrap(listTxDetail))
}
