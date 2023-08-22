package db

import (
	"github.com/Argeric/neurahive-scan-backend/util"
	"github.com/Conflux-Chain/go-conflux-sdk/types/errors"
	"gorm.io/gorm"
)

type baseStore struct {
	db *gorm.DB
}

func newBaseStore(db *gorm.DB) *baseStore {
	return &baseStore{db}
}

func (baseStore) IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, util.ErrNotFound)
}

func (bs *baseStore) Close() error {
	if mysqlDb, err := bs.db.DB(); err != nil {
		return err
	} else {
		return mysqlDb.Close()
	}
}

func (bs *baseStore) Exists(modelPtr interface{}, whereQuery string, args ...interface{}) (bool, error) {
	err := bs.db.Where(whereQuery, args...).First(modelPtr).Error
	if err == nil {
		return true, nil
	}

	if bs.IsRecordNotFound(err) {
		return false, nil
	}

	return false, err
}
