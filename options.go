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
		logger.SetIntervalSencod(5)
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
