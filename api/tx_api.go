package api

import (
	"github.com/Conflux-Chain/neurahive-scan/store"
	"github.com/gin-gonic/gin"
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
			l2TxHash:  submit.Identity,
			blockNum:  submit.BlockNumber,
			txHash:    submit.TxHash,
			address:   addrMap[submit.SenderId],
			method:    "submit",
			status:    tx.Status,
			timestamp: tx.CreatedAt.Unix(),
		}
		storageTxs = append(storageTxs, storageTx)
	}

	result := make(map[string]interface{})
	result["total"] = total
	result["list"] = storageTxs
	return result, nil
}

func listTxBrief(c *gin.Context) (interface{}, error) {
	// TODO
	return nil, nil
}

func listTxDetail(c *gin.Context) (interface{}, error) {
	// TODO
	return nil, nil
}
