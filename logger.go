package glog

import (
	"bufio"
	"fmt"
	"io/ioutil"
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

// 檔名時間格式，間隔時間類型為 Second 的設置只在開發期間使用，因此檔名時間格式精細度到分鐘即可
const FILENAMETIME string = "2006-01-02-15-04"

// 是否輸出到 Console 或 輸出成檔案，由低位到高位，以二進制分別表示
// 1. 是否輸出到 Console
// 2. 是否輸出成檔案
// 3. 是否輸出行數資訊
const TOCONSOLE int = 0b001
const TOFILE int = 0b010
const FILEINFO int = 0b100

// ====================================================================================================
// 時間轉換
// ====================================================================================================
const (
	SecondToNano int64 = 1e9
	HourToSecond int64 = 3600
	HourToNano   int64 = HourToSecond * SecondToNano
	DayToSecond  int64 = 24 * HourToSecond
	DayToNano    int64 = 24 * HourToNano
)

// ====================================================================================================
// LogLevel
// ====================================================================================================
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

// ====================================================================================================
// ShiftType
// ====================================================================================================
type ShiftType byte

const (
	ShiftNone ShiftType = iota
	ShiftSecond
	ShiftHour
	ShiftDay
	ShiftSize
	ShiftSecondAndSize
	ShiftHourAndSize
	ShiftDayAndSize
)

// ====================================================================================================
// Logger
// ====================================================================================================

type Logger struct {
	// 輸出資料夾
	folder string
	// logger 名稱
	loggerName string
	// logger 的等級
	level LogLevel
	// UTC 時區
	utc float32

	// ==================================================
	// 各個 Level 的設定
	// ==================================================
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

	// ==================================================
	// Log 換檔相關
	// ==================================================
	// 換檔類型
	shiftType ShiftType

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

// TODO: v2.0.0 時，將 callByStruct 移除
func newLogger(folder string, loggerName string, level LogLevel, callByStruct bool, options ...Option) *Logger {
	l := &Logger{
		folder:     folder,
		loggerName: loggerName,
		level:      level,
		utc:        0,
		outputs: map[LogLevel]int{
			DebugLevel: TOCONSOLE | FILEINFO,
			InfoLevel:  TOCONSOLE | FILEINFO,
			WarnLevel:  TOCONSOLE | FILEINFO,
			ErrorLevel: TOCONSOLE | FILEINFO,
		},
		writers:      make([]*bufio.Writer, 2),
		bufferSize:   4096,
		files:        make([]*os.File, 2),
		shiftType:    ShiftDay,
		timeInterval: -1,
		sizeLimit:    -1,
		cumSize:      0,
	}
	return l
}

// 可在建構子之外，設置 Logger 各項參數
func (l *Logger) SetOptions(options ...Option) {
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
}

// 設置 Log 輸出等級
func (l *Logger) SetLogLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) SetBufferSize(size uint16) {
	l.bufferSize = size
}

func (l *Logger) SetShiftCondition(shiftType ShiftType, times int64, size int64) {
	l.shiftType = shiftType
	switch shiftType {
	case ShiftSecond:
		l.setSencodInterval(times)
	case ShiftHour:
		l.setHourInterval(times)
	case ShiftDay:
		l.setDaysInterval(times)
	case ShiftSize:
		l.SetSizeLimit(size)
	case ShiftSecondAndSize:
		l.setSencodInterval(times)
		l.SetSizeLimit(size)
	case ShiftHourAndSize:
		l.setHourInterval(times)
		l.SetSizeLimit(size)
	case ShiftDayAndSize:
		l.setDaysInterval(times)
		l.SetSizeLimit(size)
	}
}

// 設置每個 Log 檔的大小，超過後更新輸出位置
func (l *Logger) SetSizeLimit(size int64) {
	l.sizeLimit = size
}

// 設置 Log 檔更新輸出位置的時間間隔，超過後更新輸出位置
func (l *Logger) setDaysInterval(days int64) {
	if days <= 0 {
		l.timeInterval = -1
		return
	} else if 105850 < days {
		l.timeInterval = days
	} else {
		l.timeInterval = 105850
	}
	now := l.getTime()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	l.date = date.Add(time.Duration(l.timeInterval * DayToNano))
}

// 設置 Log 檔更新輸出位置的時間間隔，超過後更新輸出位置
func (l *Logger) setHourInterval(hour int64) {
	if hour <= 0 {
		l.timeInterval = -1
		return
	} else if 2540400 < hour {
		l.timeInterval = hour
	} else {
		l.timeInterval = 2540400
	}
	now := l.getTime()
	date := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, nil)
	l.date = date.Add(time.Duration(l.timeInterval * HourToNano))
}

func (l *Logger) setSencodInterval(second int64) {
	if second <= 0 {
		l.timeInterval = -1
		return
	} else {
		l.timeInterval = second
	}
	l.date = l.getTime().Add(time.Duration(l.timeInterval * SecondToNano))
}

func (l *Logger) getFilePath() string {
	var filePath, timeStamp string
	// ==================================================
	// 更新時間戳
	// ==================================================
	now := l.getTime()
	switch l.shiftType {
	case ShiftDay, ShiftDayAndSize:
		t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UTC()
		timeStamp = t.Format(FILENAMETIME)
	case ShiftHour, ShiftHourAndSize:
		t := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location()).UTC()
		timeStamp = t.Format(FILENAMETIME)
	default:
		timeStamp = now.Format(FILENAMETIME)
	}
	// ==================================================
	// 根據時間戳，更新檔名
	// ==================================================
	switch l.shiftType {
	case ShiftDayAndSize, ShiftHourAndSize, ShiftSecondAndSize, ShiftSize:
		var isValidName bool
		files, _ := ioutil.ReadDir(l.folder)
		keepSearch := true
		i := 0
		fileName := fmt.Sprintf("%s-%s.log", l.loggerName, timeStamp)

		for keepSearch {
			isValidName = true

			for _, file := range files {

				if file.IsDir() {
					continue
				} else if fileName == file.Name() {
					isValidName = false
					break
				}
			}

			if isValidName {
				keepSearch = false
				filePath = path.Join(l.folder, fileName)
			} else {
				i++
				fileName = fmt.Sprintf("%s-%s-%d.log", l.loggerName, timeStamp, i)
			}
		}

	default:
		filePath = path.Join(l.folder, fmt.Sprintf("%s-%s.log", l.loggerName, timeStamp))
	}

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
	timeStamp := l.getTime().Format(DISPLAYTIME)
	var output string

	if ok {
		funcName := runtime.FuncForPC(pc).Name()
		names := strings.Split(funcName, ".")
		var label string

		if len(names) == 2 {
			label = fmt.Sprintf("[%s] %s", names[0], names[1])
		} else {
			label = fmt.Sprintf("[%s] %s", names[1], names[2])
		}

		if l.outputs[level]&FILEINFO == FILEINFO {
			message = fmt.Sprintf("%s | %s (%d)", message, file, line)
		}

		output = fmt.Sprintf("%s %s | %s | %s\n", timeStamp, level, label, message)
	} else {
		output = fmt.Sprintf("%s %s | %s\n", timeStamp, level, message)
	}

	// 是否輸出到 Console
	if l.outputs[level]&TOCONSOLE == TOCONSOLE {
		fmt.Print(output)
	}

	// 是否輸出到檔案
	if l.outputs[level]&TOFILE == TOFILE {

		// 檢查是否需要更新輸出位置
		if l.whetherNeedUpdateOutputs(output) {
			// 更新輸出位置
			err := l.updateOutput()

			if err != nil {
				return errors.Wrap(err, "更新輸出位置時發生錯誤")
			}
		}

		l.mu.Lock()
		defer l.mu.Unlock()
		size, err := l.writer.WriteString(output)

		if err != nil {
			return errors.Wrapf(err, "數據寫出時發生錯誤")
		}

		l.cumSize += int64(size)
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

	// 檢查當前路徑檔案是否已存在
	stat, err := os.Stat(filePath)

	// 若已存在
	if err == nil {
		// 更新累積檔案大小
		l.cumSize = stat.Size()
		// fmt.Printf("(l *Logger) initOutput | cumSize: %d\n", l.cumSize)
	}

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
	if l.shiftType == ShiftNone {
		// 未設置換檔條件，直接返回
		return false
	} else if l.shiftType == ShiftSize {
		// 當前大小超過已超過大小限制
		if l.cumSize >= l.sizeLimit {
			// 重置累加大小
			l.cumSize = 0
			return true
		}
		return false
	} else {
		needUpdate := l.getTime().After(l.date)

		if needUpdate {
			// fmt.Println("(l *Logger) whetherNeedUpdateOutputs | 因已達時間間隔，即將換檔")
			switch l.shiftType {
			case ShiftDay, ShiftDayAndSize:
				l.setDaysInterval(l.timeInterval)
			case ShiftHour, ShiftHourAndSize:
				l.setHourInterval(l.timeInterval)
			case ShiftSecond, ShiftSecondAndSize:
				l.setSencodInterval(l.timeInterval)
			}
		} else {
			switch l.shiftType {
			case ShiftDayAndSize, ShiftHourAndSize, ShiftSecondAndSize:
				// 當前大小超過已超過大小限制
				if l.cumSize >= l.sizeLimit {
					// fmt.Println("(l *Logger) whetherNeedUpdateOutputs | 因已達大小限制，即將換檔")
					// 重置累加大小
					l.cumSize = 0
					needUpdate = true
				}
			default:
			}
		}
		return needUpdate
	}
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

func (l *Logger) getTime() time.Time {
	var loc *time.Location
	if l.utc == -4 {
		loc, _ = time.LoadLocation("America/Nipigon")
	} else {
		loc = time.FixedZone("", int(l.utc*60*60))
	}
	return time.Now().In(loc)
}
