package glog

import "path"

var loggerMap map[string]*Logger

// TODO: v2.0.0 時，將建構子中的 callByStruct 移除
// TODO: v2.0.0 時，將建構子中的 options ...Option 移除
func init() {
	loggerMap = make(map[string]*Logger)
}

func GetLogger(folder string, loggerName string, level LogLevel, callByStruct bool, options ...Option) *Logger {
	filePath := path.Join(folder, loggerName)
	var logger *Logger
	var ok bool
	if logger, ok = loggerMap[filePath]; ok {
		return logger
	}
	logger = newLogger(folder, loggerName, level, callByStruct, options...)
	loggerMap[filePath] = logger
	return logger
}

func Flush() {
	for _, logger := range loggerMap {
		logger.Flush()
	}
}
