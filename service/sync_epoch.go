package service

import sdk "github.com/Conflux-Chain/go-conflux-sdk"

type syncConfig struct {
}

type EpochSyncer struct {
	conf *syncConfig
	cfx  sdk.ClientOperator
}
