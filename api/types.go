package api

import (
	"math/big"
	"strings"
)

type PageParam struct {
	Skip  int `form:"skip,default=0" binding:"omitempty,gte=0"`
	Limit int `form:"limit,default=10" binding:"omitempty,lte=2000"`
}

type statParam struct {
	PageParam
	MinTimestamp int    `form:"minTimestamp,default=0" binding:"omitempty,number"`
	MaxTimestamp int    `form:"maxTimestamp,default=0" binding:"omitempty,number"`
	IntervalType string `form:"intervalType,default=day" binding:"omitempty,oneof=hour day"`
	Sort         string `form:"sort,default=desc" binding:"omitempty,oneof=asc desc"`
}

func (sp *statParam) isDesc() bool {
	return strings.EqualFold(sp.Sort, "desc")
}

type txQueryParam struct {
	TxSeq *uint64 `form:"txSeq" binding:"required,number,gte=0"`
}

type StorageTx struct {
	TxSeq     uint64 `json:"txSeq"`
	BlockNum  uint64 `json:"blockNum"`
	TxHash    string `json:"txHash"`
	Address   string `json:"address"`
	Method    string `json:"method"`
	Status    uint64 `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

type TokenInfo struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

type ChargeInfo struct {
	TokenInfo `json:"tokenInfo"`
	BasicCost string `json:"basicCost"`
}

type SubmissionNode struct {
	Root   string   `json:"root"`
	Height *big.Int `json:"height"`
}

type TxBrief struct {
	TxSeq  string `json:"txSeq"`
	From   string `json:"from"`
	Method string `json:"method" default:"submit"`

	DataHash   string      `json:"dataHash"  default:""`
	DataSize   uint64      `json:"dataSize"`
	Expiration uint64      `json:"expiration"  default:"0""`
	ChargeInfo *ChargeInfo `json:"chargeInfo"`

	BlockNumber uint64 `json:"blockNumber"`
	TxHash      string `json:"txHash"`
	Timestamp   uint64 `json:"timestamp"`
	Status      uint64 `json:"status"`
	GasFee      uint64 `json:"gasFee"`
	GasUsed     uint64 `json:"gasUsed"`
	GasLimit    uint64 `json:"gasLimit"`
}

type TxDetail struct {
	TxSeq    string `json:"txSeq"`
	DataHash string `json:"dataHash"  default:""`

	StartPos    uint64            `json:"startPos"`
	EndPos      uint64            `json:"endPos"`
	PieceCounts uint64            `json:"pieceCounts"`
	Pieces      []*SubmissionNode `json:"pieces"`
}
