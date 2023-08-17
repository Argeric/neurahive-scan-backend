package store

import "time"

type transaction struct {
	Epoch             uint64     `gorm:"not null;primary_key"`
	BlockPosition     uint64     `gorm:"not null;primary_key"`
	TxPosition        uint64     `gorm:"not null;primary_key"`
	Hash              string     `gorm:"type:varchar(66);not null;index:idx_hash,length:10"`
	FromId            uint64     `gorm:"not null"`
	Nonce             uint64     `gorm:"not null"`
	ToId              uint64     `gorm:"not null"`
	DripValue         int        `gorm:"not null;default:0"`
	GasPrice          int        `gorm:"not null;default:0"`
	Gas               int        `gorm:"not null;default:0"`
	Status            int        `gorm:"not null"`
	ContractCreatedId uint64     `gorm:"not null;default:0"`
	InterfaceId       string     `gorm:"type:varchar(10);not null"`
	CreatedAt         *time.Time `gorm:"not null;index:idx_createdAt"`
}
