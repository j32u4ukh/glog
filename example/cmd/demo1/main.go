package main

import (
	"github.com/j32u4ukh/glog"
)

func main() {
	logger := glog.GetLogger("log", "not-struct", glog.DebugLevel, false)
	logger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
	// option1 := glog.BasicOption(glog.DebugLevel, false, true, true)
	// option2 := glog.BasicOption(glog.InfoLevel, false, true, true)
	// logger.SetOptions(option1, option2, glog.UtcOption(8))
	// logger.SetOptions(glog.DebugOption(), glog.UtcOption(8))
	logger.SetLogLevel(glog.DebugLevel)

	for t := 0; t < 50000; t++ {
		logger.Debug("Hello Debug! t: %d", t)
		logger.Info("Hello Info! t: %d", t)
		// time.Sleep(1 * time.Second)
	}

	print(logger)

	// obj := newObj()
	// obj.print("Hello Obj!")
	glog.Flush()

	// fi, err := os.Stat("log/not-struct-2023-01-29-00-00.log")
	// if err == nil {
	// 	fmt.Println("name:", fi.Name())
	// 	fmt.Println("size:", fi.Size())
	// 	fmt.Println("is dir:", fi.IsDir())
	// 	fmt.Println("mode::", fi.Mode())
	// 	fmt.Println("modTime:", fi.ModTime())
	// }
}

func print(logger *glog.Logger) {
	logger.Debug("print Debug!")
	logger.Info("print Info!")
}

type Obj struct {
	logger *glog.Logger
}

func newObj() *Obj {
	obj := &Obj{
		logger: glog.GetLogger("log", "Obj", glog.DebugLevel, false),
	}
	return obj
}

func (obj *Obj) print(message string) {
	obj.logger.Debug(message)
	obj.logger.Info(message)
	obj.logger.Warn(message)
	obj.logger.Error(message)
}
