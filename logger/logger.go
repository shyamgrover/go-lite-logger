package logger

import (
	"github.com/shyamgrover/go-lite-logger/logWriter"
	"github.com/shyamgrover/go-lite-logger/utils"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

type Logger struct {
	once        sync.Once            //for singleton operations
	filename    string               //logfile with complete path
	logFile     *os.File             //logFile represents an open file descriptor
	*log.Logger                      //logger instance
	logLevel    logWriter.Level      //logger log level
	status      utils.TAtomBool      //logger status..on or off
	channel     chan logWriter.Entry //log entries will go on to this channel
	stopCh      chan struct{}        //stop indicator channel for logger shutdown purposes
	worker      *logWriter.Worker    //worker that will read log entries from channel and will write to file
}

//This method initializes the channel on which log entries will go. Initiates stopChannel for signalling
// logger stop. Creates a new worker and calls worker's work method in a separate goroutine.
func (logger *Logger) init(file *os.File, errorCallback utils.ErrorFunction) {
	logger.channel = make(chan logWriter.Entry, 2048)
	logger.stopCh = make(chan struct{})
	logger.worker = logWriter.NewWorker(file, logger.channel, errorCallback)
	go logger.worker.Work()
}

//This method creates a new logger instance and returns it to the caller if success, else returns error.
// This takes logger level, logFileName,logs directory and an error callback method which is called in case of aney error.
func CreateLogger(logLevel logWriter.Level, fileName string, logDir string, errorCallback utils.ErrorFunction) (*Logger, error) {
	if len(logDir) > 0 {
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			err = os.MkdirAll(logDir, 0755)
			if err != nil {
				return nil, err
			}
		}
	} else {
		logDir = ""
	}
	filePath := logDir + fileName
	myLogger, file, err := getInstance(logLevel, filePath)
	if err == nil {
		myLogger.init(file, errorCallback)
		return myLogger, nil
	} else {
		return nil, err
	}
}

//Util method that opens a file and creates new logger instance. If success, returns logger, opened file and nil value
// for error and if error returns error to the caller and nil vaules for logger and file.
func getInstance(level logWriter.Level, filePath string) (*Logger, *os.File, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		return &Logger{
			filename: filePath,
			logLevel: level,
			status:   utils.TAtomBool{Flag: 1},
			logFile:  file,
		}, file, nil
	} else {
		return nil, nil, err
	}
}

//The method gracefully closes opened resources by logger. This can be called only once in entire logger lifecycle.
// First it closes the signalChannel. Doing this, log entries donot go on the channel. Then it waits for worker
// to close the resources. And when worker has finished closing, then it closes the logFile.
func (logger *Logger) CloseLogger() {
	logger.once.Do(func() {
		close(logger.stopCh)
		logger.worker.CloseWorker()
		logger.logFile.Close()
	})
}

// SetLevel sets the standard logger level.
func (logger *Logger) SetLevel(level logWriter.Level) {
	atomic.StoreUint32((*uint32)(&logger.logLevel), uint32(level))
}

// GetLevel returns the standard logger level.
func (logger *Logger) GetLevel() logWriter.Level {
	return logger.logLevel
}

//SetStatus sets the standard logger status. true means logging is on and false means logging is off.
func (logger *Logger) SetStatus(status bool) {
	logger.status.Set(status)
}

// GetStatus returns the standard logger status. true means logging is on and false means logging is off.
func (logger *Logger) GetStatus() bool {
	return logger.status.Get()
}

//This method returns a boolean value indicating if this particular event is loggable or not.
// It checks if log status is set to on and the given level >= the logger's level, then it returns true
// otherwise false.
func (logger *Logger) isLoggable(level logWriter.Level) bool {
	return (logger.status.Get() == true &&
		logger.logLevel >= level)
}

//This method writes log entries on to channel by checking if stop signal is received or not. If stop signal is
// received, it won't put log entries on channel else it puts entries on channel.
func (logger *Logger) logEntry(level logWriter.Level, args ...interface{}) {
	select {
	case <-logger.stopCh:
		return
	default:
		entry := logWriter.NewEntry(level, args)
		logger.channel <- entry
	}
}

//This method is similar to logEntry method but takes format as an argument as well.
func (logger *Logger) logFormattedEntry(level logWriter.Level, format string, args ...interface{}) {
	select {
	case <-logger.stopCh:
		return
	default:
		entry := logWriter.NewFormattedEntry(logWriter.DebugLevel, format, args)
		logger.channel <- entry
	}
}

// Debug logs a message at level Debug on the standard logger. This takes variadic interface type
// arguments, checks if the event is loggable and writes it to the channel.
// If not loggable, method simply returns.
func (logger *Logger) Debug(args ...interface{}) {
	if logger.isLoggable(logWriter.DebugLevel) {
		logger.logEntry(logWriter.DebugLevel, args)
	}
}

// Info logs a message at level Info on the standard logger. This takes variadic interface type
// arguments, checks if the event is loggable and writes it to the channel.
// If not loggable, method simply returns.
func (logger *Logger) Info(args ...interface{}) {
	if logger.isLoggable(logWriter.InfoLevel) {
		logger.logEntry(logWriter.InfoLevel, args)
	}
}

// Warn logs a message at level Warn on the standard logger. This takes variadic interface type
// arguments, checks if the event is loggable and writes it to the channel.
// If not loggable, method simply returns.
func (logger *Logger) Warn(args ...interface{}) {
	if logger.isLoggable(logWriter.WarnLevel) {
		logger.logEntry(logWriter.WarnLevel, args)
	}
}

// Error logs a message at level Error on the standard logger. This takes variadic interface type
// arguments, checks if the event is loggable and writes it to the channel.
// If not loggable, method simply returns.
func (logger *Logger) Error(args ...interface{}) {
	if logger.isLoggable(logWriter.ErrorLevel) {
		logger.logEntry(logWriter.ErrorLevel, args)
	}
}

// Debugf logs a message at level Debug on the standard logger. This takes format and variadic interface
// type arguments, checks if the event is loggable and writes it to the channel.
// If not loggable, method simply returns.
func (logger *Logger) Debugf(format string, args ...interface{}) {
	if logger.isLoggable(logWriter.DebugLevel) {
		logger.logFormattedEntry(logWriter.DebugLevel, format, args)
	}
}

// Infof logs a message at level Info on the standard logger. This takes format and variadic interface
// type arguments, checks if the event is loggable and writes it to the channel.
// If not loggable, method simply returns.
func (logger *Logger) Infof(format string, args ...interface{}) {
	if logger.isLoggable(logWriter.InfoLevel) {
		logger.logFormattedEntry(logWriter.InfoLevel, format, args)
	}
}

// Warnf logs a message at level Warn on the standard logger. This takes format and variadic interface
// type arguments, checks if the event is loggable and writes it to the channel.
// If not loggable, method simply returns.
func (logger *Logger) Warnf(format string, args ...interface{}) {
	if logger.isLoggable(logWriter.WarnLevel) {
		logger.logFormattedEntry(logWriter.WarnLevel, format, args)
	}
}

// Errorf logs a message at level Error on the standard logger. This takes format and variadic interface
// type arguments, checks if the event is loggable and writes it to the channel.
// If not loggable, method simply returns.
func (logger *Logger) Errorf(format string, args ...interface{}) {
	if logger.isLoggable(logWriter.ErrorLevel) {
		logger.logFormattedEntry(logWriter.ErrorLevel, format, args)
	}
}

// Debugfunc logs a message at level Debug on the standard logger. This takes variadic function
// type arguments(that return string values). It checks if the event is loggable then,
// executes the functions and creates entry from variadic interface type values and writes
// entry to the channel. If not loggable, method simply returns.
func (logger *Logger) Debugfunc(args ...utils.FunctionArg) {
	if logger.isLoggable(logWriter.DebugLevel) {
		var loggerArgs = make([]interface{}, 0, 50)
		for _, argument := range args {
			loggerArgs = append(loggerArgs, argument())
		}
		logger.logEntry(logWriter.DebugLevel, loggerArgs)
	}
}

// Infofunc logs a message at level Info on the standard logger. This takes variadic function
// type arguments(that return string values). It checks if the event is loggable then,
// executes the functions and creates entry from variadic interface type values and writes
// entry to the channel. If not loggable, method simply returns.
func (logger *Logger) Infofunc(args ...utils.FunctionArg) {
	if logger.isLoggable(logWriter.InfoLevel) {
		var loggerArgs = make([]interface{}, 0, 50)
		for _, argument := range args {
			loggerArgs = append(loggerArgs, argument())
		}
		logger.logEntry(logWriter.InfoLevel, loggerArgs)
	}
}

// Warnfunc logs a message at level Warn on the standard logger. This takes variadic function
// type arguments(that return string values). It checks if the event is loggable then,
// executes the functions and creates entry from variadic interface type values and writes
// entry to the channel. If not loggable, method simply returns.
func (logger *Logger) Warnfunc(args ...utils.FunctionArg) {
	if logger.isLoggable(logWriter.WarnLevel) {
		var loggerArgs = make([]interface{}, 0, 50)
		for _, argument := range args {
			loggerArgs = append(loggerArgs, argument())
		}
		logger.logEntry(logWriter.WarnLevel, loggerArgs)
	}
}

// Errorfunc logs a message at level Error on the standard logger. This takes variadic function
// type arguments(that return string values). It checks if the event is loggable then,
// executes the functions and creates entry from variadic interface type values and writes
// entry to the channel. If not loggable, method simply returns.
func (logger *Logger) Errorfunc(args ...utils.FunctionArg) {
	if logger.isLoggable(logWriter.ErrorLevel) {
		var loggerArgs = make([]interface{}, 0, 50)
		for _, argument := range args {
			loggerArgs = append(loggerArgs, argument())
		}
		logger.logEntry(logWriter.ErrorLevel, loggerArgs)
	}
}
