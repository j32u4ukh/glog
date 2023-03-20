package glog

import "fmt"

type Option interface {
	SetOption(*Logger)
}

/*
&^: AND NOT 運算
0 AND NOT 0     0
0 AND NOT 1     0
1 AND NOT 0     1
1 AND NOT 1     0
*/

type basicOption struct {
	Level     LogLevel
	ToConsole bool
	ToFile    bool
	FileInfo  bool
}

func BasicOption(level LogLevel, toConsole bool, toFile bool, fileInfo bool) *basicOption {
	o := &basicOption{
		Level:     level,
		ToConsole: toConsole,
		ToFile:    toFile,
		FileInfo:  toFile,
	}
	return o
}

func (o *basicOption) SetOption(logger *Logger) {
	state := logger.outputs[o.Level]

	if o.ToConsole {
		state |= TOCONSOLE
	} else {
		state &^= TOCONSOLE
	}

	if o.ToFile {
		state |= TOFILE
		logger.SetShiftCondition(ShiftDayAndSize, 1, 5*MB)
		fmt.Printf("(o *basicOption) SetOption | 5 MB: %d\n", 5*MB)
	} else {
		state &^= TOFILE
	}

	if o.FileInfo {
		state |= FILEINFO
	} else {
		state &^= FILEINFO
	}

	logger.outputs[o.Level] = state
}

type defaultOption struct {
	debugToFile bool
	infoToFile  bool
}

func DefaultOption(debugToFile bool, infoToFile bool) *defaultOption {
	o := &defaultOption{
		debugToFile: debugToFile,
		infoToFile:  infoToFile,
	}
	return o
}

func (o *defaultOption) SetOption(logger *Logger) {
	if o.debugToFile {
		logger.outputs[DebugLevel] |= TOFILE
	} else {
		logger.outputs[DebugLevel] &^= TOFILE
	}

	if o.infoToFile {
		logger.outputs[InfoLevel] |= TOFILE
	} else {
		logger.outputs[InfoLevel] &^= TOFILE
	}

	logger.outputs[WarnLevel] |= TOFILE
	logger.outputs[ErrorLevel] |= TOFILE
	logger.SetShiftCondition(ShiftDayAndSize, 1, 10*MB)
}

type utcOption struct {
	/*
		UTC(WET - 歐洲西部時區，GMT - 格林威治)
		UTC+01:00(CET - 歐洲中部時區)	UTC-01:00(CVT - 維德角)
		UTC+02:00(EET - 歐洲東部時區)	UTC-02:00(FNT - 費爾南多·迪諾羅尼亞群島)
		UTC+03:00(MSK - 莫斯科時區)		UTC-03:00(BRT - 巴西利亞)
		UTC+03:30(IRST - 伊朗)			UTC-03:30(NST - 紐芬蘭島)
		UTC+04:00(GST - 海灣)			UTC-04:00(AST - 大西洋)
		UTC+04:30(AFT - 阿富汗)
		UTC+05:00(PKT - 巴基斯坦)		UTC-05:00(EST - 北美東部)
		UTC+05:30(IST - 印度)
		UTC+05:45(NPT - 尼泊爾)
		UTC+06:00(BHT - 孟加拉)			UTC-06:00(CST - 北美中部)
		UTC+06:30(MMT - 緬甸)
		UTC+07:00(ICT - 中南半島)		UTC-07:00(MST - 北美山區)
		UTC+08:00(CT/CST - 中原)		UTC-08:00(PST - 太平洋)
		UTC+09:00(JST - 日本)			UTC-09:00(AKST - 阿拉斯加)
		UTC+09:30(ACST - 澳洲中部)		UTC-09:30(MIT - 馬克薩斯群島)
		UTC+10:00(AEST - 澳洲東部)		UTC-10:00(HST - 夏威夷-阿留申)
		UTC+10:30(LHST - 豪勳爵群島)
		UTC+11:00(VUT - 萬那杜)			UTC-11:00(SST - 美屬薩摩亞)
		UTC+12:00(NZST - 紐西蘭)		UTC-12:00(IDLW - 國際換日線)
		UTC+12:45(CHAST - 查塔姆群島)
		UTC+13:00(PHOT - 菲尼克斯群島)
		UTC+14:00(LINT - 萊恩群島)
	*/
	utc float32
}

func UtcOption(utc float32) *utcOption {
	o := &utcOption{
		utc: utc,
	}
	return o
}

func (o *utcOption) SetOption(logger *Logger) {
	if o.utc < -12 {
		o.utc = -12
	} else if o.utc > 14 {
		o.utc = 14
	}

	logger.setUtc(o.utc)
}

type folderOption struct {
	folder string
}

func FolderOption(folder string) *folderOption {
	o := &folderOption{
		folder: folder,
	}
	return o
}

func (o *folderOption) SetOption(logger *Logger) {
	logger.folder = o.folder
}

type _debugOption struct {
}

func debugOption() *_debugOption {
	o := &_debugOption{}
	return o
}

func (o *_debugOption) SetOption(logger *Logger) {
	logger.outputs[DebugLevel] = TOCONSOLE | TOFILE | FILEINFO | LINEINFO
	logger.outputs[InfoLevel] = TOCONSOLE | TOFILE | FILEINFO | LINEINFO
	logger.SetShiftCondition(ShiftSecondAndSize, 30, 2*KB)
}
