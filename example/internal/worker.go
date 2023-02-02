package internal

import (
	"time"

	"github.com/j32u4ukh/glog"
)

var logger *glog.Logger

func Init(lg *glog.Logger) {
	logger = lg
	logger.Info("Init internal package.")
}

func Run() {
	for {
		logger.Debug("Run in internal package.")
		time.Sleep(500 * time.Millisecond)
		logger.Warn("Run in internal package.")
		time.Sleep(500 * time.Millisecond)
		logger.Error("Run in internal package.")
		time.Sleep(500 * time.Millisecond)
	}
}
