package store

import (
	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"gorm.io/gorm"
	"strings"
	"time"
)

type Address struct {
	Id        uint64     `gorm:"primary_key"`
	Hex       string     `gorm:"type:varchar(40);unique:idx_hex"`
	BlockTime *time.Time `gorm:"not null;index:idx_blockTime,sort:desc"`
	CreatedAt *time.Time
}

func (Address) TableName() string {
	return "addresses"
}

type AddressStore struct {
	*mysql.Store
}

func newAddressStore(db *gorm.DB) *AddressStore {
	return &AddressStore{
		Store: mysql.NewStore(db),
	}
}

func (as *AddressStore) Add(dbTx *gorm.DB, data string, blockTime *time.Time) (uint64, error) {
	hex := strings.ToLower(strings.TrimPrefix(data, "0x"))

	var addr Address
	existed, err := as.Store.Exists(&addr, "hex = ?", hex) //TODO using LRU cache for improving the query performance
	if err != nil {
		return 0, err
	}
	if existed {
		return addr.Id, nil
	}

	addr = Address{
		Hex:       hex,
		BlockTime: blockTime,
	}
	if dbTx == nil {
		dbTx = as.Store.DB
	}
	if err := dbTx.Create(&addr).Error; err != nil {
		return 0, err
	}

	return addr.Id, nil
}

// MapAddrIdToHex TODO LRU cache
func (as *AddressStore) MapAddrIdToHex(addrIds []uint64) (map[uint64]string, error) {
	addresses := new([]Address)
	err := as.DB.Raw("select * from addresses where id in ?", addrIds).Scan(addresses).Error
	if err != nil {
		return nil, err
	}

	m := make(map[uint64]string)
	for _, addr := range *addresses {
		m[addr.Id] = addr.Hex
	}

	return m, nil
}
