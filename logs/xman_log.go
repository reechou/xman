// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_log.go

package xmanlog

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const DATEFORMAT = "2006-01-02"

type UNIT int64

const (
	_       = iota
	KB UNIT = 1 << (iota * 10)
	MB
	GB
	TB
)

// log level
const (
	LOG_DEFAULT = iota
	LOG_ERROR
	LOG_DEBUG
	LOG_NETWORK
	LOG_ALL
)

// log level strings
var levels = []string{
	"[DEFAULT]",
	"[ERROR]",
	"[DEBUG]",
	"[NETWORK]",
	"[ALL]",
}

type LoggerInfo struct {
	logLevel int32
	maxFileSize int64
	maxFileCount int32
	dailyRolling bool
	ifConsoleShow bool
	RollingFile bool
	logObj *LogFileInfo
}

type LogFileInfo struct {
	dir string
	filename string
	fileSuffix int
	isCover bool
	fileDate *time.Time
	mutex *sync.RWMutex
	logfile *os.File
	lg *log.Logger
}

var LogInfo LoggerInfo

func init() {
	LogInfo.logLevel = LOG_DEFAULT
	LogInfo.dailyRolling = true
	LogInfo.ifConsoleShow = false
	LogInfo.RollingFile = false
}

func SetConsoleShow(ifConsole bool) {
	LogInfo.ifConsoleShow = ifConsole
}

func SetLogLevel(level int32) {
	LogInfo.logLevel = level
}

// uint for KB MB GB TB
// file size = maxSize * uint
func SetRollingFile(fileDir, fileName string, maxCount int32, maxSize int64, unit UNIT) {
	LogInfo.maxFileCount = maxCount
	LogInfo.maxFileSize = maxSize * int64(unit)
	LogInfo.RollingFile = true
	LogInfo.dailyRolling = false
	LogInfo.logObj = &LogFileInfo{dir: fileDir, filename: fileName, isCover: false, mutex: new(sync.RWMutex)}
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	for i := 1; i <= int(maxCount); i++ {
		if ifExistFile(fileDir + "/" + fileName + "." + strconv.Itoa(i)) {
			LogInfo.logObj.fileSuffix = i
		} else {
			break
		}
	}
	if !LogInfo.logObj.ifMustRename() {
		LogInfo.logObj.logfile, _ = os.OpenFile(fileDir + "/" + fileName, os.O_RDWR | os.O_APPEND | os.O_CREATE, 0666)
		LogInfo.logObj.lg = log.New(LogInfo.logObj.logfile, "", log.Ldate | log.Ltime | log.Lshortfile)
	} else {
		LogInfo.logObj.rename()
	}
	go fileMonitor()
}

func SetRollingDaily(fileDir, fileName string) {
	LogInfo.RollingFile = false
	LogInfo.dailyRolling = true
	t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
	LogInfo.logObj = &LogFileInfo{dir: fileDir, filename: fileName, fileDate: &t, isCover: false, mutex: new(sync.RWMutex)}
	LogInfo.logObj.mutex.Lock()
	defer  LogInfo.logObj.mutex.Unlock()
	if !LogInfo.logObj.ifMustRename() {
		LogInfo.logObj.logfile, _ = os.OpenFile(fileDir + "/" + fileName, os.O_RDWR | os.O_APPEND | os.O_CREATE, 0666)
		LogInfo.logObj.lg = log.New(LogInfo.logObj.logfile, "", log.Ldate | log.Ltime | log.Lshortfile)
	} else {
		LogInfo.logObj.rename()
	}
}

func console(s ...interface{}) {
	if LogInfo.ifConsoleShow {
		_, file, line, _ := runtime.Caller(2)
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		log.Println(file + ":" + strconv.Itoa(line), s)
	}
}

func catchError() {
	if err := recover(); err != nil {
		log.Println("err", err)
	}
}

func Log(logLevel int32, v ...interface{}) {
	if LogInfo.logObj.lg == nil {
		log.Println("LogInfo.logObj.lg == nil")
		return
	}
	if LogInfo.dailyRolling {
		fileCheck()
	}
	defer catchError()
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	if LogInfo.logLevel >= logLevel {
		LogInfo.logObj.lg.Output(2, fmt.Sprint(levels[logLevel], v))
		console(levels[logLevel], v)
	}
}

func Logf(logLevel int32, format string, a ...interface{}) {
	if LogInfo.logObj.lg == nil {
		log.Println("LogInfo.logObj.lg == nil")
		return
	}
	if LogInfo.dailyRolling {
		fileCheck()
	}
	defer catchError()
	LogInfo.logObj.mutex.Lock()
	defer LogInfo.logObj.mutex.Unlock()
	if LogInfo.logLevel >= logLevel {
		format = levels[logLevel] + format
		LogInfo.logObj.lg.Output(2, fmt.Sprintf(format, a...))
//		console(levels[logLevel], format, a)
	}
}

func (logFile *LogFileInfo) ifMustRename() bool {
	if LogInfo.dailyRolling {
		t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
		if t.After(*logFile.fileDate) {
			return true
		}
	} else {
		if LogInfo.maxFileCount > 1 {
			if getFileSize(logFile.dir + "/" + logFile.filename) >= LogInfo.maxFileSize {
				return true
			}
		}
	}

	return false
}

func (logFile *LogFileInfo) rename() {
	if LogInfo.dailyRolling {
		fn := logFile.dir + "/" + logFile.filename + "." + logFile.fileDate.Format(DATEFORMAT)
		if !ifExistFile(fn) && logFile.ifMustRename() {
			if logFile.logfile != nil {
				logFile.logfile.Close()
			}
			err := os.Rename(logFile.dir + "/" + logFile.filename, fn)
			if err != nil {
				logFile.lg.Println("rename error, ", err.Error())
			}
			t, _ := time.Parse(DATEFORMAT, time.Now().Format(DATEFORMAT))
			logFile.fileDate = &t
			logFile.logfile, _ = os.Create(logFile.dir + "/" + logFile.filename)
			logFile.lg = log.New(LogInfo.logObj.logfile, "", log.Ldate | log.Ltime | log.Lshortfile)
		}
	} else {
		logFile.coverNextOne()
	}
}

func (logFile *LogFileInfo) coverNextOne() {
	logFile.fileSuffix = logFile.nextSuffix()
	if logFile.logfile != nil {
		logFile.logfile.Close()
	}
	newFileName := logFile.dir + "/" + logFile.filename + "." + strconv.Itoa(int(logFile.fileSuffix))
	if ifExistFile(newFileName) {
		os.Remove(newFileName)
	}
	os.Rename(logFile.dir + "/" + logFile.filename, newFileName)
	logFile.logfile, _ = os.Create(logFile.dir + "/" + logFile.filename)
	logFile.lg = log.New(LogInfo.logObj.logfile, "", log.Ldate | log.Ltime | log.Lshortfile)
}

func (logFile *LogFileInfo) nextSuffix() int {
	return int(logFile.fileSuffix % int(LogInfo.maxFileCount) + 1)
}

func getFileSize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		fmt.Println(e.Error())
		return 0
	}
	return f.Size()
}

func ifExistFile(file string) bool {
	_, err := os.Stat(file)
	return err != nil || os.IsExist(err)
}

func fileMonitor() {
	timer := time.NewTicker(1 * time.Second)
	for {
		select {
		case <- timer.C:
			fileCheck()
		}
	}
}

func fileCheck() {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	if LogInfo.logObj != nil && LogInfo.logObj.ifMustRename() {
		LogInfo.logObj.mutex.Lock()
		defer LogInfo.logObj.mutex.Unlock()
		LogInfo.logObj.rename()
	}
}
