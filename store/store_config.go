package store

import (
	"github.com/Conflux-Chain/go-conflux-util/store/mysql"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CfgDataUplinkRateInterval     = "dataUplinkRateInterval"
	DefaultDataUplinkRateInterval = "1m"
	CfgDataUplinkRate             = "dataUplinkRate"
)

type Config struct {
	Name    string `gorm:"type:varchar(32);primary_key"`
	Content string `gorm:"type:varchar(128)"`
}

func (Config) TableName() string {
	return "configs"
}

type ConfigStore struct {
	*mysql.Store
}

func newConfigStore(db *gorm.DB) *ConfigStore {
	return &ConfigStore{
		Store: mysql.NewStore(db),
	}
}

func (cs *ConfigStore) Add(name, content string) error {
	return cs.DB.Create(&Config{
		Name:    name,
		Content: content,
	}).Error
}

func (cs *ConfigStore) Upsert(name, content string) error {
	return cs.DB.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&Config{
		Name:    name,
		Content: content,
	}).Error
}

func (cs *ConfigStore) Get(name string) (*string, error) {
	var cfg Config
	exist, err := cs.Exists(&cfg, "name = ?", CfgDataUplinkRate)
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, errors.New("Data uplink rate not stat.")
	}

	return &cfg.Content, nil
}
