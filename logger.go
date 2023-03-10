package glog

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// 時間輸出格式
const DISPLAYTIME string = "2006/01/02 15:04:05"

// 檔名時間格式
const FILENAMETIME string = "2006-01-02-15-04-05"

//
const TOCONSOLE int = 0b001
const TOFILE int = 0b010
const FILEINFO int = 0b100

// log level
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "Debug"
	case InfoLevel:
		return "Info "
	case WarnLevel:
		return "Warn "
	case ErrorLevel:
		return "Error"
	default:
		return "Unknown"
	}
}

const (
	SecondToNano int64 = 1e9
	HourToSecond int64 = 3600
	HourToNano   int64 = HourToSecond * SecondToNano
	DayToNano    int64 = 24 * HourToNano
)

type Logger struct {
	// 輸出資料夾
	folder string
	// logger 名稱
	loggerName string
	// logger 的等級
	level LogLevel

	// ==================================================
	// 各個 Level 的設定
	// ==================================================
	// 是否輸出到 Console 或 輸出成檔案，由低位到高位，以二進制分別表示
	// 1. 是否輸出到 Console
	// 2. 是否輸出成檔案
	// 3. 是否輸出行數資訊
	outputs map[LogLevel]int

	// ==================================================
	// 數據輸出用
	// ==================================================
	// 當前寫出數據用 Writer
	writer *bufio.Writer
	// 管理兩個 Writer，用於換檔時交替用
	writers []*bufio.Writer
	// 初始化 Writer 的緩衝大小
	bufferSize uint16
	// 管理兩個 File，用於換檔時交替用
	files []*os.File
	// 讀寫鎖
	mu sync.RWMutex

	// ===== Log 時間管理 =====
	// Log 檔更新輸出位置的時間間隔(單位：小時)，超過後更新輸出位置
	// time.Duration 的上限為 2540400 小時，超過的話直接設為 2540400
	timeInterval int64
	// 換檔時間戳
	date time.Time

	// ===== Log 檔案大小管理 =====
	// 每個 Log 檔的大小限制，超過後更新輸出位置
	sizeLimit int64
	// 累計輸出行數，每 CheckLines 行再檢查一次 Log 檔的大小限制是否超出
	cumSize int64
}

func newLogger(folder string, loggerName string, level LogLevel, callByStruct bool, options ...Option) *Logger {
	l := &Logger{
		folder:     folder,
		loggerName: loggerName,
		level:      level,
		outputs: map[LogLevel]int{
			DebugLevel: 1,
			InfoLevel:  1,
			WarnLevel:  0b111,
			ErrorLevel: 0b111,
		},
		writers:      make([]*bufio.Writer, 2),
		bufferSize:   4096,
		files:        make([]*os.File, 2),
		timeInterval: -1,
		sizeLimit:    -1,
		cumSize:      0,
	}

	// 根據各個 Option 調整 Logger 參數
	for _, option := range options {
		option.SetOption(l)
	}

	// 檢查是否有輸出到檔案需求，若有，則檢查輸出資料夾是否存在。若資料夾不存在，則產生。
	for _, state := range l.outputs {
		if state&TOFILE == TOFILE {
			l.initOutput()
			break
		}
	}
	return l
}

// 設置 Log 輸出等級
func (l *Logger) SetLogLevel(level LogLevel) {
	l.level = level
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
	l.date = date.Add(time.Duration(l.timeInterval * HourToNano))
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
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, nil).UTC()
	l.date = date.Add(time.Duration(l.timeInterval * DayToNano))
}

func (l *Logger) SetIntervalSencod(second int64) {
	if second <= 0 {
		l.timeInterval = -1
		return
	} else {
		l.timeInterval = second
	}
	l.date = time.Now().Add(time.Duration(l.timeInterval * SecondToNano))
}

func (l *Logger) getFilePath() string {
	timeStamp := time.Now().Format(FILENAMETIME)
	filePath := path.Join(l.folder, fmt.Sprintf("%s-%s.log", l.loggerName, timeStamp))
	return filePath
}

func (l *Logger) Debug(message string, a ...any) {
	l.Logout(DebugLevel, fmt.Sprintf(message, a...))
}

func (l *Logger) Info(message string, a ...any) {
	l.Logout(InfoLevel, fmt.Sprintf(message, a...))
}

func (l *Logger) Warn(message string, a ...any) {
	l.Logout(WarnLevel, fmt.Sprintf(message, a...))
}

func (l *Logger) Error(message string, a ...any) {
	l.Logout(ErrorLevel, fmt.Sprintf(message, a...))
}

func (l *Logger) Logout(level LogLevel, message string) error {
	if l.level > level {
		return nil
	}

	pc, file, line, ok := runtime.Caller(2)
	timeStamp := time.Now().Format(DISPLAYTIME)
	var output string

	if ok {
		funcName := runtime.FuncForPC(pc).Name()
		names := strings.Split(funcName, ".")

		if len(names) == 2 {
			if l.outputs[level]&FILEINFO == FILEINFO {
				output = fmt.Sprintf("%s %s | [%s] %s | %s | %s (%d)\n",
					timeStamp, level, l.loggerName, names[1], message, file, line)
			} else {
				output = fmt.Sprintf("%s %s | [%s] %s | %s\n",
					timeStamp, level, l.loggerName, names[1], message)
			}
		} else {
			if l.outputs[level]&FILEINFO == FILEINFO {
				output = fmt.Sprintf("%s %s | [%s] %s | %s | %s (%d)\n",
					timeStamp, level, l.loggerName, names[2], message, file, line)
			} else {
				output = fmt.Sprintf("%s %s | [%s] %s | %s\n",
					timeStamp, level, l.loggerName, names[2], message)
			}
		}
	} else {
		output = fmt.Sprintf("%s %s | %s\n", timeStamp, level, message)
	}

	// 是否輸出到 Console
	if l.outputs[level]&TOCONSOLE == TOCONSOLE {
		fmt.Print(output)
	}

	// 是否輸出到檔案
	if l.outputs[level]&TOFILE == TOFILE {
		l.mu.Lock()
		l.writer.WriteString(output)
		l.mu.Unlock()

		// 檢查是否需要更新輸出位置
		if l.whetherNeedUpdateOutputs(output) {
			// 更新輸出位置
			err := l.updateOutput()

			if err != nil {
				return errors.Wrap(err, "更新輸出位置時發生錯誤")
			}
		}
	}
	return nil
}

// 初始化輸出結構
func (l *Logger) initOutput() error {
	_, err := os.Stat(l.folder)

	if err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(l.folder, os.ModePerm)
		}
	}

	filePath := l.getFilePath()
	l.files[0], err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		return errors.Wrapf(err, "開啟輸出檔時發生錯誤, path: %s\n", l.getFilePath())
	}

	l.writers[0] = bufio.NewWriterSize(l.files[0], int(l.bufferSize))
	l.writer = l.writers[0]
	return nil
}

// 檢查是否需要更換輸出檔
func (l *Logger) whetherNeedUpdateOutputs(output string) bool {
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

		if l.cumSize >= l.sizeLimit {
			needUpdate = true
			l.cumSize = 0
		}
	}

	return needUpdate
}

// 更新輸出位置
func (l *Logger) updateOutput() error {
	var err error
	idx := -1

	if l.files[0] == nil {
		idx = 0
	} else if l.files[1] == nil {
		idx = 1
	} else {
		return errors.Wrap(err, "切換輸出檔時發生錯誤")
	}

	newPath := l.getFilePath()
	l.files[idx], err = os.OpenFile(newPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		return errors.Wrapf(err, "開啟輸出檔時發生錯誤, path: %s\n", newPath)
	}

	l.writers[idx] = bufio.NewWriterSize(l.files[idx], int(l.bufferSize))
	l.writer = l.writers[idx]

	// 清空並關閉另一組 logger
	idx = 1 - idx
	if l.writers[idx].Buffered() > 0 {
		l.writers[idx].Flush()
	}
	l.writers[idx] = nil
	l.files[idx].Close()
	l.files[idx] = nil
	return nil
}

func (l *Logger) Flush() {
	var idx int
	for idx, l.writer = range l.writers {
		if l.files[idx] != nil {
			if (l.writer != nil) && (l.writer.Buffered() > 0) {
				l.writer.Flush()
			}
			l.files[idx].Close()
		}
	}
}
