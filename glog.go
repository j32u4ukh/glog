package glog

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

// log level
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarningLevel
	ErrorLevel
)

const (
	NanoToDay    int64 = 24 * 3600 * 1e9
	NanoToHour   int64 = 3600 * 1e9
	NanoToSecond int64 = 1e9
	CheckLines   uint8 = 5
)

type Logger struct {
	outConsole bool
	outFile    bool
	writer     *bufio.Writer
	buffers    []*bufio.Writer
	bufferSize uint16
	files      []*os.File
	mu         sync.RWMutex
	exitChan   chan os.Signal
	root       string
	fileName   string
	level      LogLevel
	// Log 檔更新輸出位置的時間間隔(單位：小時)，超過後更新輸出位置
	// time.Duration 的上限為 2540400 小時，超過的話直接設為 2540400
	timeInterval int64
	date         time.Time
	// 每個 Log 檔的大小限制，超過後更新輸出位置
	sizeLimit int64
	// 累計輸出行數，每 CheckLines 行再檢查一次 Log 檔的大小限制是否超出
	cumSize int64
}

func NewLogger(outConsole bool) *Logger {
	l := &Logger{
		outConsole:   outConsole,
		buffers:      make([]*bufio.Writer, 2),
		bufferSize:   4096,
		files:        make([]*os.File, 2),
		exitChan:     make(chan os.Signal),
		level:        DebugLevel,
		timeInterval: -1,
		sizeLimit:    -1,
		cumSize:      0,
	}
	return l
}

// 設置 Log 輸出等級
func (l *Logger) SetLogLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) SetOutput(root string, fileName string) error {
	l.outFile = true
	l.root = root
	l.fileName = fileName
	_, err := os.Stat(root)

	if err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(root, os.ModePerm)
		} else {
			return errors.Wrapf(err, "Log root error, root: %s\n", root)
		}
	}

	out := l.getFilePath()
	l.files[0], err = os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		return errors.Wrapf(err, "Open log file err: %s\n", out)
	}

	l.buffers[0] = bufio.NewWriterSize(l.files[0], int(l.bufferSize))
	l.writer = l.buffers[0]
	signal.Notify(l.exitChan, os.Interrupt, syscall.SIGTERM /*, os.Kill*/)
	// 退出時的處理
	go l.exitHandle()
	return nil
}

// 設置每個 Log 檔的大小，超過後更新輸出位置
func (l *Logger) SetSizeLimit(size int64) {
	l.sizeLimit = size
}

func (l *Logger) SetBufferSize(size uint16) {
	l.bufferSize = size
}

// 設置 Log 檔更新輸出位置的時間間隔，超過後更新輸出位置
func (l *Logger) SetHourInterval(hour int32) {
	if hour <= 0 {
		l.timeInterval = -1
		return
	} else if 2540400 < hour {
		l.timeInterval = int64(hour)
	} else {
		l.timeInterval = 2540400
	}

	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, nil).UTC()
	l.date = date.Add(time.Duration(l.timeInterval * NanoToHour))
}

// 設置 Log 檔更新輸出位置的時間間隔，超過後更新輸出位置
func (l *Logger) SetDaysInterval(days int32) {
	if days <= 0 {
		l.timeInterval = -1
		return
	} else if 105850 < days {
		l.timeInterval = int64(days)
	} else {
		l.timeInterval = 105850
	}

	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.UTC().Location())
	l.date = date.Add(time.Duration(l.timeInterval * NanoToHour))
}

func (l *Logger) SetIntervalSencod(second int64) {
	if second <= 0 {
		l.timeInterval = -1
		return
	} else {
		l.timeInterval = second
	}
	l.date = time.Now().Add(time.Duration(l.timeInterval * NanoToSecond))
}

func (l *Logger) getFilePath() string {
	timeStamp := time.Now().Format("2006-01-02-15-04")
	filePath := path.Join(l.root, fmt.Sprintf("%s-%s.log", l.fileName, timeStamp))
	return filePath
}

// 退出時的處理
func (l *Logger) exitHandle() {
	<-l.exitChan
	fmt.Printf("[Logger] exitHandle | %s\n", l.fileName)

	if l.writer.Buffered() > 0 {
		l.writer.Flush()
	}

	if l.files[0] != nil {
		l.files[0].Close()
	}

	if l.files[1] != nil {
		l.files[1].Close()
	}

	os.Exit(1)
}

func (l *Logger) Debug(message string) {
	if l.level > DebugLevel {
		return
	}

	l.Logout("Debug", message)
}

func (l *Logger) Debugf(format string, a ...any) {
	if l.level > DebugLevel {
		return
	}

	l.Logout("Debug", fmt.Sprintf(format, a...))
}

func (l *Logger) Info(message string) {
	if l.level > InfoLevel {
		return
	}

	l.Logout("Info", message)
}

func (l *Logger) Infof(format string, a ...any) {
	if l.level > InfoLevel {
		return
	}

	l.Logout("Info", fmt.Sprintf(format, a...))
}

func (l *Logger) Warning(message string) {
	if l.level > WarningLevel {
		return
	}

	l.Logout("Warning", message)
}

func (l *Logger) Warningf(format string, a ...any) {
	if l.level > WarningLevel {
		return
	}

	l.Logout("Warning", fmt.Sprintf(format, a...))
}

func (l *Logger) Error(message string) {
	l.Logout("Error", message)
}

func (l *Logger) Errorf(format string, a ...any) {
	l.Logout("Error", fmt.Sprintf(format, a...))
}

func (l *Logger) Logout(level string, message string) {
	pc, file, line, ok := runtime.Caller(2)
	now := time.Now()
	// 2022/08/26 10:48:32
	timeStamp := now.Format("2006/01/02 15:04:05.0700")
	var output string

	if ok {
		// fileName := path.Base(file)
		funcName := runtime.FuncForPC(pc).Name()
		names := strings.Split(funcName, ".")

		if len(names) == 2 {
			output = fmt.Sprintf("%s %s | [%s] %s | %s\n%s (%d)\n",
				timeStamp, level, names[0], names[1], message, file, line)
		} else {
			output = fmt.Sprintf("%s %s | [%s] %s | %s\n%s (%d)\n",
				timeStamp, level, names[1], names[2], message, file, line)
		}
	} else {
		output = fmt.Sprintf("%s %s | %s\n", timeStamp, level, message)
	}

	if l.outConsole {
		fmt.Println(output)
	}

	l.mu.Lock()
	l.writer.WriteString(output)
	l.mu.Unlock()

	// 檢查是否需要更新輸出位置
	if l.WhetherNeedUpdateOutputs(output) {
		// 更新輸出位置
		l.updateOutput()
	}
}

func (l *Logger) WhetherNeedUpdateOutputs(output string) bool {
	needUpdate := false

	if l.timeInterval != -1 {
		needUpdate = time.Now().After(l.date)

		if needUpdate {
			l.SetIntervalSencod(l.timeInterval)
		}
	}

	if l.sizeLimit != -1 {
		size := len(output)
		l.cumSize += int64(size)
		fmt.Printf("Size: %d, l.cumSize: %d\n", size, l.cumSize)

		if l.cumSize >= l.sizeLimit {
			needUpdate = true
			l.cumSize = 0
		}
	}

	return needUpdate
}

func (l *Logger) updateOutput() {
	idx := -1

	if l.files[0] == nil {
		idx = 0
	} else if l.files[1] == nil {
		idx = 1
	}

	var err error
	l.files[idx], err = os.OpenFile(l.getFilePath(), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		log.Fatalf("Open log file err: %+v\n", err)
	}

	l.buffers[idx] = bufio.NewWriterSize(l.files[idx], int(l.bufferSize))
	l.writer = l.buffers[idx]

	// 清空並關閉另一組 logger
	idx = 1 - idx
	if l.buffers[idx].Buffered() > 0 {
		l.buffers[idx].Flush()
	}
	l.buffers[idx] = nil
	l.files[idx].Close()
	l.files[idx] = nil
}
