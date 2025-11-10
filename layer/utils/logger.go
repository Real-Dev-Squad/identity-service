package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

type Logger struct {
	service       string
	env           string
	contextFields map[string]interface{}
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

func NewLogger(service string) *Logger {
	env := os.Getenv("environment")
	if env == "" {
		env = "DEVELOPMENT"
	}
	return &Logger{
		service:       service,
		env:           env,
		contextFields: make(map[string]interface{}),
	}
}

func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}, err error) {
	mergedFields := make(map[string]interface{})
	for k, v := range l.contextFields {
		mergedFields[k] = v
	}
	for k, v := range fields {
		mergedFields[k] = v
	}
	
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     string(level),
		Service:   l.service,
		Message:   message,
		Fields:    mergedFields,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	jsonBytes, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		fmt.Printf("[%s] %s: %s", level, l.service, message)
		if err != nil {
			fmt.Printf(" - Error: %v", err)
		}
		if len(fields) > 0 {
			fmt.Printf(" - Fields: %+v", fields)
		}
		fmt.Println()
		return
	}

	fmt.Println(string(jsonBytes))
}

func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	mergedFields := mergeFields(fields...)
	l.log(LogLevelDebug, message, mergedFields, nil)
}

func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	mergedFields := mergeFields(fields...)
	l.log(LogLevelInfo, message, mergedFields, nil)
}

func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	mergedFields := mergeFields(fields...)
	l.log(LogLevelWarn, message, mergedFields, nil)
}

func (l *Logger) WarnWithError(message string, err error, fields ...map[string]interface{}) {
	mergedFields := mergeFields(fields...)
	l.log(LogLevelWarn, message, mergedFields, err)
}

func (l *Logger) Error(message string, err error, fields ...map[string]interface{}) {
	mergedFields := mergeFields(fields...)
	l.log(LogLevelError, message, mergedFields, err)
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		service:       l.service,
		env:           l.env,
		contextFields: make(map[string]interface{}),
	}
	for k, v := range l.contextFields {
		newLogger.contextFields[k] = v
	}
	for k, v := range fields {
		newLogger.contextFields[k] = v
	}
	return newLogger
}

func mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}

var defaultLogger *Logger

func init() {
	defaultLogger = NewLogger("identity-service")
}

func GetLogger() *Logger {
	return defaultLogger
}

func LogDebug(message string, fields ...map[string]interface{}) {
	defaultLogger.Debug(message, fields...)
}

func LogInfo(message string, fields ...map[string]interface{}) {
	defaultLogger.Info(message, fields...)
}

func LogWarn(message string, fields ...map[string]interface{}) {
	defaultLogger.Warn(message, fields...)
}

func LogWarnWithError(message string, err error, fields ...map[string]interface{}) {
	defaultLogger.WarnWithError(message, err, fields...)
}

func LogError(message string, err error, fields ...map[string]interface{}) {
	defaultLogger.Error(message, err, fields...)
}
