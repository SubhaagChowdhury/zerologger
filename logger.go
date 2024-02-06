package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reg_proc/config"
	"strconv"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const loggingTimeFormat = "02-Jan-2006 15:04:05"

// ANSI color escape codes
var (
	colorReset = "\033[0m"
	// colorGray   = "\033[90m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	// colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	// colorCyan   = "\033[36m"
	// colorWhite  = "\033[37m"
)

var levelColorMap = map[string]string{"DEBUG": colorYellow, "INFO": colorGreen, "WARN": colorPurple, "ERROR": colorRed}

type Object struct {
	logFilePath string
	logFileName string
	logLevel    string // default global logging level

	Logger zerolog.Logger // responsible for writing logs
	config *config.Configurations
}

func (logger *Object) ModuleInit(cfgObj *config.Configurations) bool {
	logger.config = cfgObj
	logger.logFilePath = logger.config.LoggerConfigDetails.LogFilePath
	logger.logFileName = logger.config.LoggerConfigDetails.LogFileName
	logger.logLevel = logger.config.LoggerConfigDetails.LogLevel

	switch strings.ToUpper(logger.logLevel) {
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "WARNING":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		fmt.Println("Incorrect log level given : " + logger.logLevel)
		return false
	}

	zerolog.TimeFieldFormat = loggingTimeFormat     // TimeFieldFormat defines the time format of the Time field type.
	zerolog.MessageFieldName = "Message"            // Can be set to customize message field name.
	zerolog.LevelFieldName = "LoggingLevel"         // Can be set to customize logging level field name.
	zerolog.TimestampFieldName = "LoggingTimestamp" // Can be set to customize logging timestamp field name.

	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string { // this specifies the file which called the logging statement.
		short := strings.Split(filepath.Base(file), ".")[0]
		file = short

		applicationName := logger.config.ApplicationDetails.ApplicationName

		return fmt.Sprintf("[[%s] ==> %s:%s]", applicationName, file, strconv.Itoa(line)) // application name, file and line of invocation
	}

	var (
		currentLogLevel string
		writers         []io.Writer
	)

	consoleOutput := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: loggingTimeFormat} // for writing the log message to the console

	consoleOutput.FormatLevel = func(i interface{}) string { // logging level will be displayed like --  [INFO]
		currentLogLevel = strings.ToUpper(fmt.Sprintf("%v", i))
		return levelColorMap[currentLogLevel] + strings.ToUpper(fmt.Sprintf("[%v]", i)) + colorReset
	}
	consoleOutput.FormatMessage = func(i interface{}) string { // message will be displayed like -- example: [message]
		return levelColorMap[currentLogLevel] + fmt.Sprintf("[%v]", i) + colorReset
	}
	consoleOutput.FormatTimestamp = func(i interface{}) string { // formatting how time should be displayed -- example: [08-Jul-2022 08:02:11]
		return fmt.Sprintf("[%v]", i)
	}

	writers = append(writers, consoleOutput, logger.fileLogger())

	mw := io.MultiWriter(writers...)

	logger.Logger = zerolog.New(mw).With().Timestamp().Caller().Logger()
	return true
}

// fileLogger method is used to log a message to the file
func (logger *Object) fileLogger() io.Writer {
	_, err := os.Stat(logger.logFilePath)
	if os.IsNotExist(err) { // if folder does not exist, create it
		if err = os.MkdirAll(logger.logFilePath, 0o777); err != nil {
			log.Error().Err(err).Str("path", logger.logFilePath).Msg("can't create log directory")

			return nil
		}
	}

	directory := logger.logFilePath
	fileName := logger.logFileName + ".log."
	timePattern := "%Y-%m-%d_%H"

	completeLogFileName := fmt.Sprintf("%s%s", directory, fileName+timePattern)

	writer, err := rotatelogs.New(
		completeLogFileName,
		rotatelogs.WithRotationTime(time.Hour*1),
		rotatelogs.WithMaxAge(7*24*365*3*time.Hour),
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to Initialise Log File")

		return nil
	}

	return writer
}
