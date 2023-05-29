package glog

import (
	"bufio"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type void struct{}

var null void

// 時間輸出格式
const DISPLAYTIME string = "2006/01/02 15:04:05"

// 檔名時間格式，間隔時間類型為 Second 的設置只在開發期間使用，因此檔名時間格式精細度到分鐘即可
const FILENAMETIME string = "2006-01-02-15-04"

// 是否輸出到 Console 或 輸出成檔案，由低位到高位，以二進制分別表示
// 1. 是否輸出到 Console
// 2. 是否輸出成檔案
// 3. 是否輸出檔案資訊
// 3. 是否輸出行數資訊
const TOCONSOLE int = 0b0001
const TOFILE int = 0b0010
const FILEINFO int = 0b0100
const LINEINFO int = 0b1000

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
// 檔案大小轉換(單位：Byte)
// ====================================================================================================
const (
	KB int64 = 1024
	MB int64 = 1024 * KB
	GB int64 = 1024 * MB
	TB int64 = 1024 * GB
	PB int64 = 1024 * TB
	EB int64 = 1024 * PB
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

func (st ShiftType) String() string {
	switch st {
	case ShiftSecond:
		return "ShiftSecond"
	case ShiftHour:
		return "ShiftHour"
	case ShiftDay:
		return "ShiftDay"
	case ShiftSize:
		return "ShiftSize"
	case ShiftSecondAndSize:
		return "ShiftSecondAndSize"
	case ShiftHourAndSize:
		return "ShiftHourAndSize"
	case ShiftDayAndSize:
		return "ShiftDayAndSize"
	default:
		return "None"
	}
}

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
	loc *time.Location
	utc float32

	// ==================================================
	// 各個 Level 的設定
	// ==================================================
	outputs map[LogLevel]int

	// ==================================================
	// 數據輸出用
	// ==================================================
	// 將 log 檔的建立，延後至第一筆輸出之前
	outputInited bool
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
	// 換檔索引值
	nShift int32
	// 每個 Log 檔的大小限制，超過後更新輸出位置
	sizeLimit int64
	// 累計輸出行數，每 CheckLines 行再檢查一次 Log 檔的大小限制是否超出
	cumSize int64
}

func newLogger(loggerName string, level LogLevel, options ...Option) *Logger {
	l := &Logger{
		folder:     "",
		loggerName: loggerName,
		level:      level,
		loc:        time.UTC,
		utc:        0,
		outputs: map[LogLevel]int{
			DebugLevel: TOCONSOLE | LINEINFO,
			InfoLevel:  TOCONSOLE | LINEINFO,
			WarnLevel:  TOCONSOLE | FILEINFO | LINEINFO,
			ErrorLevel: TOCONSOLE | FILEINFO | LINEINFO,
		},
		outputInited: false,
		writers:      make([]*bufio.Writer, 2),
		bufferSize:   4096,
		files:        make([]*os.File, 2),
		shiftType:    ShiftDay,
		timeInterval: 0,
		nShift:       0,
		sizeLimit:    0,
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
}

// 設置 Log 輸出等級
func (l *Logger) SetLogLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) SetFolder(folder string) {
	l.folder = folder
}

func (l *Logger) SetBufferSize(size uint16) {
	l.bufferSize = size
}

func (l *Logger) SetShiftCondition(shiftType ShiftType, times int64, size int64) {
	// 重置累加大小
	l.cumSize = 0

	// 設置換檔類型
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
	default:
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
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, l.loc)
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
	date := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, l.loc)
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
		var label, pkg string
		var temp []string

		if len(names) == 2 {
			temp = strings.Split(names[0], "/")
			pkg = temp[len(temp)-1]
			label = fmt.Sprintf("[%s] %s", pkg, names[1])
		} else {
			temp = strings.Split(names[1], "/")
			pkg = temp[len(temp)-1]
			label = fmt.Sprintf("[%s] %s", pkg, names[2])
		}

		if l.outputs[level]&FILEINFO == FILEINFO {
			message = fmt.Sprintf("%s | %s", message, file)
		}

		if l.outputs[level]&LINEINFO == LINEINFO {
			message = fmt.Sprintf("%s | (%d)", message, line)
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
		if l.outputInited {
			status := l.whetherNeedUpdateOutputs()

			// 檢查是否需要更新輸出位置
			if status != 0 {
				// 更新輸出位置
				err := l.updateOutput(status)

				if err != nil {
					return errors.Wrap(err, "更新輸出位置時發生錯誤")
				}
			}
		} else {
			err := l.initOutput()

			if err != nil {
				return errors.Wrap(err, "Failed to initialize output.")
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

// 可使用 runtime.FuncForPC(ptr) 獲得進一步的資訊
func (l *Logger) CheckCaller(skip int) uintptr {
	ptr, file, line, ok := runtime.Caller(skip)
	fmt.Printf("(l *Logger) CheckCaller | skip: %d, pc: %d, file: %s, line: %d, ok: %v\n", skip, ptr, file, line, ok)

	if ok {
		return ptr
	} else {
		return 0
	}
}

func (l *Logger) CheckCallers() {
	var pc uintptr
	var file string
	var line, skip int = 0, 0
	ok := true

	for ok {
		pc, file, line, ok = runtime.Caller(skip)

		if ok {
			fmt.Printf("(l *Logger) CheckCaller | skip: %d, pc: %d, file: %s, line: %d, ok: %v\n", skip, pc, file, line, ok)
			skip++
		}
	}
}

// 初始化輸出結構
func (l *Logger) initOutput() error {
	if l.folder == "" {
		return errors.New("未定義輸出資料夾")
	}

	_, err := os.Stat(l.folder)

	if err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(l.folder, os.ModePerm)
		}
	}

	filePath := l.getInitPath()
	fmt.Printf("(l *Logger) initOutput | cumSize: %d, filePath: %s\n", l.cumSize, filePath)

	l.files[0], err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		return errors.Wrapf(err, "開啟輸出檔時發生錯誤, path: %s\n", filePath)
	}

	l.writers[0] = bufio.NewWriterSize(l.files[0], int(l.bufferSize))
	l.writer = l.writers[0]
	l.outputInited = true
	return nil
}

func (l *Logger) getFilePath() string {
	var filePath string
	// ==================================================
	// 更新時間戳
	// ==================================================
	timeStamp := l.getFileTime()

	// ==================================================
	// 根據時間戳，更新檔名
	// ==================================================
	switch l.shiftType {
	case ShiftDayAndSize, ShiftHourAndSize, ShiftSecondAndSize, ShiftSize:
		var fileName string

		if l.nShift == 0 {
			fmt.Printf("當前時間區段內首次取得路徑\n")
			fileName = fmt.Sprintf("%s-%s.log", l.loggerName, timeStamp)
		} else {
			fileName = fmt.Sprintf("%s-%s-%d.log", l.loggerName, timeStamp, l.nShift)
			fmt.Printf("第 %d 次取得路徑, fileName: %s\n", l.nShift, fileName)
			l.nShift++
		}

		filePath = path.Join(l.folder, fileName)

	default:
		filePath = path.Join(l.folder, fmt.Sprintf("%s-%s.log", l.loggerName, timeStamp))
	}

	return filePath
}

func (l *Logger) getInitPath() string {
	files, _ := ioutil.ReadDir(l.folder)
	names := map[string]void{}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		names[file.Name()] = null
		fmt.Printf("(l *Logger) getInitPath | Existed file: %s\n", file.Name())
	}

	timeStamp := l.getFileTime()
	fileName := fmt.Sprintf("%s-%s.log", l.loggerName, timeStamp)
	var filePath string
	var stat fs.FileInfo
	var err error

	// 檔名尚未存在，表示可以使用
	if _, ok := names[fileName]; !ok {
		filePath = path.Join(l.folder, fileName)
		l.nShift = 1
		return filePath
	}

	for {
		fileName = fmt.Sprintf("%s-%s-%d.log", l.loggerName, timeStamp, l.nShift+1)

		if _, ok := names[fileName]; !ok {
			break
		}

		l.nShift++
	}

	if l.nShift == 0 {
		fileName = fmt.Sprintf("%s-%s.log", l.loggerName, timeStamp)
	} else {
		fileName = fmt.Sprintf("%s-%s-%d.log", l.loggerName, timeStamp, l.nShift)
	}

	filePath = path.Join(l.folder, fileName)
	fmt.Printf("(l *Logger) getInitPath | filePath1: %s\n", filePath)
	stat, err = os.Stat(filePath)

	// 若該檔名已存在
	if err == nil {
		// 更新累積檔案大小
		l.cumSize = stat.Size()
		status := l.whetherNeedUpdateOutputs()
		fmt.Printf("(l *Logger) getInitPath | status: %d, cumSize: %d, filePath: %s\n", status, l.cumSize, filePath)

		// 若已達換檔達條件
		if status != 0 {
			l.cumSize = 0
			l.nShift++
			fileName = fmt.Sprintf("%s-%s-%d.log", l.loggerName, timeStamp, l.nShift)
			filePath = path.Join(l.folder, fileName)
		}
	}

	l.nShift++
	fmt.Printf("(l *Logger) getInitPath | nShift: %d, cumSize: %d, filePath2: %s\n", l.nShift, l.cumSize, filePath)
	return filePath
}

func (l *Logger) getFileTime() string {
	var timeStamp string
	now := l.getTime()
	switch l.shiftType {
	case ShiftDay, ShiftDayAndSize:
		t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, l.loc)
		timeStamp = t.Format(FILENAMETIME)
	case ShiftHour, ShiftHourAndSize:
		t := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, l.loc)
		timeStamp = t.Format(FILENAMETIME)
	default:
		timeStamp = now.Format(FILENAMETIME)
	}
	return timeStamp
}

// 檢查是否需要更換輸出檔(0: 無須換檔; 1: 已達大小限制; 2: 已達時間間隔)
func (l *Logger) whetherNeedUpdateOutputs() byte {
	if l.shiftType == ShiftNone {
		// 未設置換檔條件，直接返回
		return 0
	} else if l.shiftType == ShiftSize {
		// 當前大小 是否已超過 大小限制
		if l.cumSize >= l.sizeLimit {
			// fmt.Printf("(l *Logger) whetherNeedUpdateOutputs | shiftType: %s, cumSize: %d, 因已達大小限制(%d)，即將換檔\n",
			// 	l.shiftType, l.cumSize, l.sizeLimit)
			return 1
		}
		return 0
	} else {
		if l.getTime().After(l.date) {
			// fmt.Printf("(l *Logger) whetherNeedUpdateOutputs | shiftType: %s, 因已達時間間隔，即將換檔", l.shiftType)
			return 2
		} else {
			switch l.shiftType {
			case ShiftDayAndSize, ShiftHourAndSize, ShiftSecondAndSize:
				// 當前大小 是否已超過 大小限制
				if l.cumSize >= l.sizeLimit {
					// fmt.Printf("(l *Logger) whetherNeedUpdateOutputs | shiftType: %s, cumSize: %d, 因已達大小限制(%d)，即將換檔\n",
					// 	l.shiftType, l.cumSize, l.sizeLimit)
					return 1
				}
			default:
			}
			return 0
		}
	}
}

// 更新輸出位置
func (l *Logger) updateOutput(status byte) error {
	l.cumSize = 0

	// 超過時間間隔限制
	if status == 2 {
		l.nShift = 0
		l.SetShiftCondition(l.shiftType, l.timeInterval, l.sizeLimit)
	}

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

func (l *Logger) setUtc(utc float32) {
	l.utc = utc

	if l.utc == -4 {
		l.loc, _ = time.LoadLocation("America/Nipigon")
	} else {
		l.loc = time.FixedZone("", int(l.utc*60*60))
	}
}

func (l *Logger) getTime() time.Time {
	return time.Now().In(l.loc)
}
