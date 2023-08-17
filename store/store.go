package store

import config2 "neurahive-scan-backend/config"

type MysqlStore struct {
	*baseStore

	config *config2.Config
}
