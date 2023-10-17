package api

import (
	"github.com/Conflux-Chain/neurahive-scan/store"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"strconv"
	"time"
)

func listTx(c *gin.Context) (interface{}, error) {
	var pageP PageParam
	if err := c.ShouldBind(&pageP); err != nil {
		return nil, err
	}

	submits := new([]store.Submit)
	total, err := db.List(db.DB.Model(&store.Submit{}), true, pageP.Skip, pageP.Limit, submits)
	if err != nil {
		return nil, err
	}

	addrIds := make([]uint64, 0)
	txHashes := make([]string, 0)
	for _, submit := range *submits {
		addrIds = append(addrIds, submit.SenderId)
		txHashes = append(txHashes, submit.TxHash)
	}
	addrMap, err := db.MapAddrIdToHex(addrIds)
	if err != nil {
		return nil, err
	}
	txMap, err := db.MapTxHashToTx(txHashes)
	if err != nil {
		return nil, err
	}

	storageTxs := make([]StorageTx, 0)
	for _, submit := range *submits {
		tx := txMap[submit.TxHash]
		storageTx := StorageTx{
			TxSeq:     submit.SubmissionIndex,
			BlockNum:  submit.BlockNumber,
			TxHash:    submit.TxHash,
			Address:   addrMap[submit.SenderId],
			Method:    "submit",
			Status:    tx.Status,
			Timestamp: tx.CreatedAt.Unix(),
		}
		storageTxs = append(storageTxs, storageTx)
	}

	result := make(map[string]interface{})
	result["total"] = total
	result["list"] = storageTxs
	return result, nil
}

func getTxBrief(c *gin.Context) (interface{}, error) {
	var param txQueryParam
	if err := c.ShouldBind(&param); err != nil {
		return nil, err
	}

	var submit struct {
		SubmissionIndex  uint64
		SubmissionLength uint64
		SenderId         uint64
		BlockNumber      uint64
		Hash             string
		CreatedAt        *time.Time
		Value            *decimal.Decimal
		Status           uint64
		GasFee           uint64
		GasUsed          uint64
		GasLimit         uint64
	}

	err := db.DB.Raw(`select s.submission_index, s.submission_length, s.sender_id, t.block_number, t.hash, 
       t.created_at, t.status, t.gas_fee, t.gas_used, t.gas_limit, ts.value from submits s 
       left join txs t on s.tx_hash = t.hash 
       left join erc20_transfers ts on t.hash = ts.tx_hash 
       where s.submission_index =?`, param.TxSeq).Take(&submit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Errorf("Record not found, txSeq %v", *param.TxSeq)
	}
	if err != nil {
		logrus.WithError(err).Error("Failed to query databases")
		return nil, errors.Errorf("Biz error, txSeq %v", *param.TxSeq)
	}

	addrIds := []uint64{submit.SenderId}
	addrMap, err := db.MapAddrIdToHex(addrIds)
	if err != nil {
		return nil, err
	}

	result := TxBrief{
		TxSeq: strconv.FormatUint(submit.SubmissionIndex, 10),
		From:  "0x" + addrMap[submit.SenderId],

		DataSize: submit.SubmissionLength,
		ChargeInfo: &ChargeInfo{
			TokenInfo: *chargeToken,
			BasicCost: submit.Value.String(),
		},

		BlockNumber: submit.BlockNumber,
		TxHash:      "0x" + submit.Hash,
		Timestamp:   uint64(submit.CreatedAt.Unix()),
		Status:      submit.Status,
		GasFee:      submit.GasFee,
		GasUsed:     submit.GasUsed,
		GasLimit:    submit.GasLimit,
	}

	return result, nil
}

func getTxDetail(c *gin.Context) (interface{}, error) {
	var param txQueryParam
	if err := c.ShouldBind(&param); err != nil {
		return nil, err
	}

	var submit store.Submit
	exist, err := db.Exists(&submit, "submission_index = ?", param.TxSeq)
	if err != nil {
		logrus.WithError(err).Error("Failed to query databases")
		return nil, errors.Errorf("Biz error, txSeq %v", *param.TxSeq)
	}
	if !exist {
		return nil, errors.Errorf("Record not found, txSeq %v", *param.TxSeq)
	}

	nodes, err := getSubmitEvent("0x" + submit.TxHash)
	if err != nil {
		return nil, err
	}

	result := TxDetail{
		TxSeq:       strconv.FormatUint(submit.SubmissionIndex, 10),
		StartPos:    submit.StartPos,
		EndPos:      submit.StartPos + submit.Length,
		PieceCounts: submit.Nodes,
		Pieces:      nodes,
	}

	return result, nil
}
