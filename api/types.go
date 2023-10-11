package api

import "strings"

type PageParam struct {
	Skip  int `form:"skip,default:0" binding:"gte=0" `
	Limit int `form:"limit,default:10" binding:"lte=2000"`
}

type statParam struct {
	PageParam
	MinTimestamp int    `form:"minTimestamp,default:0" binding:"number" `
	MaxTimestamp int    `form:"maxTimestamp,default:0" binding:"number"`
	IntervalType string `form:"intervalType,default:day" binding:"oneof=hour day" `
	Sort         string `form:"sort,default:desc" binding:"oneof=asc desc" `
}

func (sp *statParam) isDesc() bool {
	return strings.EqualFold(sp.Sort, "desc")
}

type StorageTx struct {
	l2TxHash  string
	blockNum  uint64
	txHash    string
	address   string
	method    string
	status    uint64
	timestamp int64
}
