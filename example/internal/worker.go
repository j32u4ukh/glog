package internal

import (
	"time"

	"github.com/j32u4ukh/glog"
)

var worker *Worker

type Worker struct {
	logger *glog.Logger
}

func (w *Worker) Debug(format string, args ...any) {
	w.logger.Debug(format, args...)
}

func (w *Worker) Info(format string, args ...any) {
	w.logger.Info(format, args...)
}

func (w *Worker) Warn(format string, args ...any) {
	w.logger.Warn(format, args...)
}

func (w *Worker) Error(format string, args ...any) {
	w.logger.Error(format, args...)
}

func Init(lg *glog.Logger) {
	worker = &Worker{
		logger: lg,
	}
	worker.Info("Init internal package.")
}

func Run() {
	for {
		worker.Debug("Run in internal package.")
		time.Sleep(300 * time.Millisecond)
		worker.Warn("Run in internal package.")
		time.Sleep(300 * time.Millisecond)
		worker.Error("Run in internal package.")
		time.Sleep(300 * time.Millisecond)
	}
}
