package store

import (
	"database/sql"
	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"github.com/openweb3/web3go/types"
	"gorm.io/gorm"
	"time"
)

type Block struct {
	BlockNumber uint64     `gorm:"primary_key;autoIncrement:false"`
	Hash        string     `gorm:"type:varchar(64);not null;index:idx_hash,length:10"`
	CreatedAt   *time.Time `gorm:"not null;index:idx_block_time,sort:desc"`
}

func NewBlock(data *types.Block) *Block {
	blockTime := time.Unix(int64(data.Timestamp), 0)
	return &Block{
		BlockNumber: data.Number.Uint64(),
		Hash:        data.Hash.String()[2:],
		CreatedAt:   &blockTime,
	}
}

func (Block) TableName() string {
	return "blocks"
}

type blockStore struct {
	baseStore *mysql.Store
}

func newBlockStore(db *gorm.DB) *blockStore {
	return &blockStore{
		baseStore: mysql.NewStore(db),
	}
}

func (bs *blockStore) Add(dbTx *gorm.DB, block *Block) error {
	return dbTx.Create(block).Error
}

func (bs *blockStore) MaxBlock() (uint64, bool, error) {
	var maxBlock sql.NullInt64

	db := bs.baseStore.DB.Model(&Block{}).Select("MAX(block_number)")
	if err := db.Find(&maxBlock).Error; err != nil {
		return 0, false, err
	}

	if !maxBlock.Valid {
		return 0, false, nil
	}

	return uint64(maxBlock.Int64), true, nil
}

func (bs *blockStore) BlockHash(blockNumber uint64) (string, bool, error) {
	var blk Block

	existed, err := bs.baseStore.Exists(&blk, "block_number = ?", blockNumber)
	if err != nil {
		return "", false, err
	}

	return blk.Hash, existed, nil
}

func (bs *blockStore) Pop(dbTx *gorm.DB, block uint64) error {
	return dbTx.Where("block_number >= ?", block).Delete(&Block{}).Error
}
