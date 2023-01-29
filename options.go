package glog

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
		logger.SetDaysInterval(1)
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

	if o.debugToFile || o.infoToFile {
		logger.SetDaysInterval(1)
	}
}
