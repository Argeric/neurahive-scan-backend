package api

import (
	"encoding/hex"
	"github.com/Conflux-Chain/go-conflux-util/api"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	nhContract "github.com/Conflux-Chain/neurahive-scan/contract"
	"github.com/Conflux-Chain/neurahive-scan/store"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/openweb3/web3go"
	"github.com/sirupsen/logrus"
	"strings"
)

var (
	sdk           *web3go.Client
	db            *store.MysqlStore
	chargeToken   *TokenInfo
	flowAddr      string
	flowSubmitSig string
)

func MustInit(client *web3go.Client, store *store.MysqlStore) {
	sdk = client
	db = store

	var charge struct {
		Erc20TokenAddress string
	}
	viperutil.MustUnmarshalKey("charge", &charge)

	name, symbol, decimals, err := nhContract.TokenInfo(client, charge.Erc20TokenAddress)
	if err != nil {
		logrus.WithError(err).Fatal("Get erc20 token info")
	}

	chargeToken = &TokenInfo{
		Address:  charge.Erc20TokenAddress,
		Name:     name,
		Symbol:   symbol,
		Decimals: decimals,
	}

	var flow struct {
		Address              string
		SubmitEventSignature string
	}
	viperutil.MustUnmarshalKey("flow", &flow)
	flowAddr = flow.Address
	flowSubmitSig = flow.SubmitEventSignature
}

func RegisterRouter(router *gin.Engine) {
	apiRoute := router.Group("/api")

	statRoute := apiRoute.Group("/statistic")
	statRoute.GET("transaction/list", api.Wrap(listTxStat))
	statRoute.GET("storage/list", api.Wrap(listDataStat))
	statRoute.GET("cost/basic/list", api.Wrap(listBasicCostStat))

	txRoute := apiRoute.Group("/transaction")
	txRoute.GET("list", api.Wrap(listTx))
	txRoute.GET("brief", api.Wrap(getTxBrief))
	txRoute.GET("detail", api.Wrap(getTxDetail))
}

func getSubmitEvent(hash string) ([]*SubmissionNode, error) {
	rcpt, err := sdk.Eth.TransactionReceipt(common.HexToHash(hash))
	if err != nil {
		return nil, err
	}

	var nodes []*SubmissionNode
	for _, log := range rcpt.Logs {
		addr := log.Address.String()
		sig := log.Topics[0].String()
		if !strings.EqualFold(addr, flowAddr) || sig != flowSubmitSig {
			continue
		}

		flowSubmit, err := nhContract.DummyFlowFilterer().ParseSubmit(*log.ToEthLog())
		if err != nil {
			return nil, err
		}

		nodeArr := flowSubmit.Submission.Nodes
		for _, n := range nodeArr {
			node := SubmissionNode{
				Root:   "0x" + hex.EncodeToString(n.Root[:]),
				Height: n.Height,
			}
			nodes = append(nodes, &node)
		}
		break
	}

	return nodes, nil
}
