package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 结构化日志器
type Logger struct {
	*logrus.Logger
	config *Config
}

// Config 日志配置
type Config struct {
	Level      string `json:"level"`       // debug, info, warn, error
	Format     string `json:"format"`      // json, text
	Output     string `json:"output"`      // stdout, file, both
	File       string `json:"file"`        // 日志文件路径
	MaxSize    int    `json:"max_size"`    // 单个文件最大大小(MB)
	MaxBackups int    `json:"max_backups"` // 保留的旧文件数量
	MaxAge     int    `json:"max_age"`     // 保留的天数
	Compress   bool   `json:"compress"`    // 是否压缩旧文件
}

// Fields 日志字段类型
type Fields map[string]interface{}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		File:       "logs/xbox.log",
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}
}

// NewLogger 创建新的日志器
func NewLogger(config *Config) (*Logger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	logger := logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %s", config.Level)
	}
	logger.SetLevel(level)

	// 设置日志格式
	switch config.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
				logrus.FieldKeyFunc:  "function",
				logrus.FieldKeyFile:  "file",
			},
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	default:
		return nil, fmt.Errorf("invalid log format: %s", config.Format)
	}

	// 设置输出
	var output io.Writer
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "file":
		if err := ensureLogDir(config.File); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %v", err)
		}
		output = &lumberjack.Logger{
			Filename:   config.File,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}
	case "both":
		if err := ensureLogDir(config.File); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %v", err)
		}
		fileWriter := &lumberjack.Logger{
			Filename:   config.File,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}
		output = io.MultiWriter(os.Stdout, fileWriter)
	default:
		return nil, fmt.Errorf("invalid log output: %s", config.Output)
	}

	logger.SetOutput(output)

	// 添加调用者信息
	logger.SetReportCaller(true)

	return &Logger{
		Logger: logger,
		config: config,
	}, nil
}

// ensureLogDir 确保日志目录存在
func ensureLogDir(logFile string) error {
	dir := filepath.Dir(logFile)
	return os.MkdirAll(dir, 0755)
}

// WithFields 添加字段
func (l *Logger) WithFields(fields Fields) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields(fields))
}

// WithComponent 添加组件字段
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.Logger.WithField("component", component)
}

// WithError 添加错误字段
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// WithRequest 添加请求字段
func (l *Logger) WithRequest(method, path string) *logrus.Entry {
	return l.Logger.WithFields(logrus.Fields{
		"http_method": method,
		"http_path":   path,
	})
}

// WithAgent 添加Agent字段
func (l *Logger) WithAgent(agentID string) *logrus.Entry {
	return l.Logger.WithField("agent_id", agentID)
}

// LogRequest 记录HTTP请求
func (l *Logger) LogRequest(method, path string, status int, duration time.Duration) {
	l.WithFields(Fields{
		"http_method":   method,
		"http_path":     path,
		"http_status":   status,
		"response_time": duration.Milliseconds(),
	}).Info("HTTP request processed")
}

// LogGRPCCall 记录gRPC调用
func (l *Logger) LogGRPCCall(method string, duration time.Duration, err error) {
	entry := l.WithFields(Fields{
		"grpc_method":   method,
		"response_time": duration.Milliseconds(),
	})

	if err != nil {
		entry.WithError(err).Error("gRPC call failed")
	} else {
		entry.Info("gRPC call completed")
	}
}

// LogDatabaseOperation 记录数据库操作
func (l *Logger) LogDatabaseOperation(operation, table string, duration time.Duration, err error) {
	entry := l.WithFields(Fields{
		"db_operation":  operation,
		"db_table":      table,
		"response_time": duration.Milliseconds(),
	})

	if err != nil {
		entry.WithError(err).Error("Database operation failed")
	} else {
		entry.Debug("Database operation completed")
	}
}

// LogSystemEvent 记录系统事件
func (l *Logger) LogSystemEvent(event, component string, details Fields) {
	entry := l.WithFields(Fields{
		"event":     event,
		"component": component,
	})

	if details != nil {
		entry = entry.WithFields(logrus.Fields(details))
	}

	entry.Info("System event")
}

// LogAgentEvent 记录Agent事件
func (l *Logger) LogAgentEvent(agentID, event string, details Fields) {
	entry := l.WithFields(Fields{
		"agent_id": agentID,
		"event":    event,
	})

	if details != nil {
		entry = entry.WithFields(logrus.Fields(details))
	}

	entry.Info("Agent event")
}

// LogPerformance 记录性能指标
func (l *Logger) LogPerformance(operation string, duration time.Duration, details Fields) {
	entry := l.WithFields(Fields{
		"operation":     operation,
		"duration_ms":   duration.Milliseconds(),
		"duration_ns":   duration.Nanoseconds(),
	})

	if details != nil {
		entry = entry.WithFields(logrus.Fields(details))
	}

	if duration > time.Second {
		entry.Warn("Slow operation detected")
	} else {
		entry.Debug("Performance metric")
	}
}

// Fatal 记录致命错误并退出
func (l *Logger) Fatal(args ...interface{}) {
	l.addStackTrace().Fatal(args...)
}

// Fatalf 记录格式化致命错误并退出
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.addStackTrace().Fatalf(format, args...)
}

// Panic 记录恐慌错误并panic
func (l *Logger) Panic(args ...interface{}) {
	l.addStackTrace().Panic(args...)
}

// Panicf 记录格式化恐慌错误并panic
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.addStackTrace().Panicf(format, args...)
}

// addStackTrace 添加堆栈跟踪信息
func (l *Logger) addStackTrace() *logrus.Entry {
	// 获取调用栈信息
	pc := make([]uintptr, 10)
	n := runtime.Callers(3, pc)
	frames := runtime.CallersFrames(pc[:n])
	
	var stackTrace []string
	for {
		frame, more := frames.Next()
		stackTrace = append(stackTrace, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	
	return l.Logger.WithField("stack_trace", stackTrace)
}

// GetConfig 获取日志配置
func (l *Logger) GetConfig() *Config {
	return l.config
}

// Rotate 手动轮转日志文件
func (l *Logger) Rotate() error {
	if l.config.Output == "file" || l.config.Output == "both" {
		// 这里简化处理，实际实现需要访问lumberjack实例
		l.Info("Log rotation requested")
		return nil
	}
	return fmt.Errorf("log rotation not supported for output: %s", l.config.Output)
}