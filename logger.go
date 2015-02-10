package logger

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"time"
)

//============================= private ================================
//customize logger name

var isDebug bool
var logId string

//
const (
	_LOG_DEBUG_FLAG = log.Ldate | log.Lmicroseconds | log.Lshortfile

	_INFO_HEADER    = "[INFO]"
	_WARNING_HEADER = "[WARNING]"
	_ERROR_HEADER   = "[ERROR]"
	_DEBUG_HEADER   = "[DEBUG]"
	_FATAL_HEADER   = "[FATAL]"
)

type postFunc func(msg string, sysBuf chan string, localBuf chan string)

var accesslog *log.Logger
var errorlog *log.Logger
var accessSyslog *syslog.Writer
var errorSyslog *syslog.Writer

var post postFunc
var bufferSize = 10000
var bufMap map[string]chan string

func getPoster(debug bool) postFunc {
	return func(msg string, sysBuf chan string, localBuf chan string) {
		if !debug {
			//post to syslog
			select {
			case sysBuf <- msg:
				return
			case <-time.After(time.Millisecond * 1):
			}
		}

		//sysBuf is full or debug, so post to localBuf
		select {
		case localBuf <- msg:
			return
		case <-time.After(time.Millisecond * 1):
		}

		//anyway if localbuf is full, drop the msg
		return
	}
}

func initSyslog() error {
	var err error
	if accessSyslog, err = syslog.New(syslog.LOG_INFO|syslog.LOG_LOCAL3, logId+"_info"); err != nil {
		return err
	}
	if errorSyslog, err = syslog.New(syslog.LOG_ERR|syslog.LOG_LOCAL3, logId+"_err"); err != nil {
		return err
	}

	bufMap["InfoSys"] = make(chan string, bufferSize)
	bufMap["DebugSys"] = make(chan string, bufferSize)
	bufMap["WarningSys"] = make(chan string, bufferSize)
	bufMap["ErrorSys"] = make(chan string, bufferSize)
	bufMap["FatalSys"] = make(chan string, bufferSize)
	go syslogConsumeProc()

	return nil
}

func initLocalLog() error {
	if isDebug {
		errorlog = log.New(os.Stderr, "", _LOG_DEBUG_FLAG)
		accesslog = log.New(os.Stdout, "", _LOG_DEBUG_FLAG)
	} else {
		if access_fd, access_err := os.OpenFile(logId+"_info",
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); access_err == nil {
			accesslog = log.New(access_fd, "", _LOG_DEBUG_FLAG)
		} else {
			return access_err
		}
		if error_fd, error_err := os.OpenFile(logId+"_err",
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); error_err == nil {
			errorlog = log.New(error_fd, "", _LOG_DEBUG_FLAG)
		} else {
			return error_err
		}
	}
	bufMap["FatalLocal"] = make(chan string, bufferSize)
	bufMap["InfoLocal"] = make(chan string, bufferSize)
	bufMap["DebugLocal"] = make(chan string, bufferSize)
	bufMap["WarningLocal"] = make(chan string, bufferSize)
	bufMap["ErrorLocal"] = make(chan string, bufferSize)
	go localConsumeProc()

	return nil
}

func syslogConsumeProc() {
	defer func() {
		recover()
	}()

	for {
		select {
		case msg := <-bufMap["DebugSys"]:
			accessSyslog.Debug(msg)
		case msg := <-bufMap["InfoSys"]:
			accessSyslog.Info(msg)
		case msg := <-bufMap["WarningSys"]:
			errorSyslog.Warning(msg)
		case msg := <-bufMap["ErrorSys"]:
			errorSyslog.Err(msg)
		case msg := <-bufMap["FatalSys"]:
			errorSyslog.Crit(msg)
		}
	}
}

func localConsumeProc() {
	defer func() {
		recover()
	}()

	for {
		select {
		case msg := <-bufMap["DebugLocal"]:
			accesslog.Println(msg)
		case msg := <-bufMap["InfoLocal"]:
			accesslog.Println(msg)
		case msg := <-bufMap["WarningLocal"]:
			errorlog.Println(msg)
		case msg := <-bufMap["ErrorLocal"]:
			errorlog.Println(msg)
		case msg := <-bufMap["FatalLocal"]:
			errorlog.Println(msg)
		}
	}
}

func Init(proj string, debug bool) {
	isDebug = debug
	logId = proj
	bufMap = make(map[string]chan string, 10)
	post = getPoster(isDebug)
	if !isDebug {
		if err := initSyslog(); err != nil {
			panic(err)
		}
	}
	if err := initLocalLog(); err != nil {
		panic(err)
	}
}

//============================= private END ================================

//============================= public API ================================
func Info(v ...interface{}) {
	post(fmt.Sprint(_INFO_HEADER, fmt.Sprint(v...)), bufMap["InfoSys"], bufMap["InfoLocal"])
}

func Infof(format string, v ...interface{}) {
	post(fmt.Sprint(_INFO_HEADER, fmt.Sprintf(format, v...)), bufMap["InfoSys"], bufMap["InfoLocal"])
}

func Debug(v ...interface{}) {
	post(fmt.Sprint(_DEBUG_HEADER, fmt.Sprint(v...)), bufMap["DebugSys"], bufMap["DebugLocal"])
}

func Debugf(format string, v ...interface{}) {
	post(fmt.Sprint(_DEBUG_HEADER, fmt.Sprintf(format, v...)), bufMap["DebugSys"], bufMap["DebugLocal"])
}

func Warning(v ...interface{}) {
	post(fmt.Sprint(_WARNING_HEADER, fmt.Sprint(v...)), bufMap["WarningSys"], bufMap["WarningLocal"])
}

func Warningf(format string, v ...interface{}) {
	post(fmt.Sprint(_WARNING_HEADER, fmt.Sprintf(format, v...)), bufMap["WarningSys"], bufMap["WarningLocal"])
}

func Error(v ...interface{}) {
	post(fmt.Sprint(_ERROR_HEADER, fmt.Sprint(v...)), bufMap["ErrorSys"], bufMap["ErrorLocal"])
}

func Errorf(format string, v ...interface{}) {
	post(fmt.Sprint(_ERROR_HEADER, fmt.Sprintf(format, v...)), bufMap["ErrorSys"], bufMap["ErrorLocal"])
}

func Fatal(v ...interface{}) {
	post(fmt.Sprint(_FATAL_HEADER, fmt.Sprint(v...)), bufMap["FatalSys"], bufMap["FatalLocal"])
}

func Fatalf(format string, v ...interface{}) {
	post(fmt.Sprint(_FATAL_HEADER, fmt.Sprintf(format, v...)), bufMap["FatalSys"], bufMap["FatalLocal"])
}
