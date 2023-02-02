package internal

import "github.com/j32u4ukh/glog"

var logger *glog.Logger

func Init(lg *glog.Logger) {
	logger = lg
	logger.Info("Init internal package.")
}

func Run() {
	logger.Debug("Run in internal package.")
	logger.Warn("Run in internal package.")
	logger.Error("Run in internal package.")
}
