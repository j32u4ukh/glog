package internal

import (
	"time"

	"github.com/j32u4ukh/glog/v2"
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

func (w *Worker) run() {
	worker.Debug("Run in *Worker.")
	time.Sleep(300 * time.Millisecond)
	worker.Warn("Run in *Worker.")
	time.Sleep(300 * time.Millisecond)
	worker.Error("Run in *Worker.")
	time.Sleep(300 * time.Millisecond)
}

func Init() {
	worker = &Worker{
		logger: glog.GetLogger(0),
	}
	worker.Info("Init internal package.")
	worker.logger.SetSkip(3)
}

func Run() {
	for {
		worker.Debug("Run in internal package.")
		time.Sleep(300 * time.Millisecond)
		worker.Warn("Run in internal package.")
		time.Sleep(300 * time.Millisecond)
		worker.Error("Run in internal package.")
		time.Sleep(300 * time.Millisecond)
		worker.run()
		time.Sleep(300 * time.Millisecond)
	}
}
