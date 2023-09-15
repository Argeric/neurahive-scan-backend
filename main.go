package main

import (
	"github.com/Conflux-Chain/go-conflux-util/config"
	"github.com/Conflux-Chain/go-conflux-util/log"
	"github.com/Conflux-Chain/neurahive-scan/cmd"
)

func main() {
	config.MustInit("neurahive")
	log.MustInitFromViper()
	cmd.Execute()
}
