package util

import "github.com/pkg/errors"

// Epoch data type
type EpochDataType uint

const (
	EpochDataNil EpochDataType = iota
	EpochTransaction
	EpochLog
	EpochBlock
)

var (
	// custom errors
	ErrNotFound               = errors.New("not found")
	ErrUnsupported            = errors.New("not supported")
	ErrEpochPivotSwitched     = errors.New("epoch pivot switched")
	ErrContinousEpochRequired = errors.New("continous epoch required")
	ErrAlreadyPruned          = errors.New("data already pruned")
	ErrChainReorged           = errors.New("chain re-orged")

	// operationable epoch data types
	OpEpochDataTypes = []EpochDataType{
		EpochBlock,
		EpochTransaction,
		EpochLog,
	}
)
