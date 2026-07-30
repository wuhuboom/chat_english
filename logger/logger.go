package logger

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go-fly-muti/common"
)

const (
	RequestIDHeader = "X-Request-ID"
	RequestIDKey    = "request_id"
)

var (
	stdLogRedirectMu sync.Mutex
	undoStdLog       func()
)

// Init configures a single structured logger for the application, HTTP access
// logs and standard-library log calls.
func Init() error {
	level := zapcore.InfoLevel
	configuredLevel := strings.TrimSpace(viper.GetString("log.level"))
	if configuredLevel != "" {
		if err := level.UnmarshalText([]byte(configuredLevel)); err != nil {
			return fmt.Errorf("invalid log level %q: %w", configuredLevel, err)
		}
	}

	logPath := FilePath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	writer := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    positiveOrDefault(viper.GetInt("log.max_size"), 300),
		MaxBackups: positiveOrDefault(viper.GetInt("log.max_backups"), 7),
		MaxAge:     positiveOrDefault(viper.GetInt("log.max_age"), 30),
		Compress:   true,
		LocalTime:  true,
	}
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(fileEncoderConfig()),
		zapcore.AddSync(writer),
		level,
	)
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoderConfig()),
		zapcore.Lock(os.Stdout),
		level,
	)
	appLogger := zap.New(
		zapcore.NewTee(fileCore, consoleCore),
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	stdLogRedirectMu.Lock()
	if undoStdLog != nil {
		undoStdLog()
	}
	undoStdLog = zap.RedirectStdLog(appLogger.Named("stdlib"))
	stdLogRedirectMu.Unlock()

	zap.ReplaceGlobals(appLogger)
	zap.L().Info("logger initialized",
		zap.String("component", "system"),
		zap.String("file", logPath),
		zap.String("configured_level", level.String()),
		zap.Int("max_size_mb", writer.MaxSize),
		zap.Int("max_backups", writer.MaxBackups),
		zap.Int("max_age_days", writer.MaxAge),
	)
	return nil
}

// Sync flushes buffered log entries. It is safe to call during shutdown.
func Sync() {
	_ = zap.L().Sync()
}

// FilePath returns the configured primary application log path. Relative paths
// remain relative to the application root for backward compatibility.
func FilePath() string {
	filename := strings.TrimSpace(viper.GetString("log.filename"))
	if filename == "" {
		filename = "web_app.log"
	}
	if filepath.IsAbs(filename) {
		return filepath.Clean(filename)
	}
	root := common.RootPath
	if root == "" {
		if cwd, err := os.Getwd(); err == nil {
			root = cwd
		}
	}
	return filepath.Join(root, filename)
}

func positiveOrDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func fileEncoderConfig() zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "time"
	config.LevelKey = "level"
	config.NameKey = "logger"
	config.CallerKey = "caller"
	config.MessageKey = "message"
	config.StacktraceKey = "stack"
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncodeDuration = zapcore.MillisDurationEncoder
	config.EncodeCaller = zapcore.ShortCallerEncoder
	return config
}

func consoleEncoderConfig() zapcore.EncoderConfig {
	config := fileEncoderConfig()
	config.EncodeLevel = zapcore.CapitalLevelEncoder
	return config
}

// GinLogger records one structured access entry after each request.
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := resolveRequestID(c.GetHeader(RequestIDHeader))
		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()

		status := c.Writer.Status()
		path := c.Request.URL.Path
		if status < http.StatusBadRequest &&
			(strings.HasPrefix(path, "/static/") || path == "/system/logs") {
			return
		}
		duration := time.Since(start)
		component := "http"
		if c.Request.Method != http.MethodGet &&
			c.Request.Method != http.MethodHead &&
			c.Request.Method != http.MethodOptions {
			component = "audit"
		}
		fields := []zap.Field{
			zap.String("component", component),
			zap.String("request_id", requestID),
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("route", c.FullPath()),
			zap.String("query", sanitizeQuery(c.Request.URL.Query())),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("response_bytes", c.Writer.Size()),
			zap.Duration("duration", duration),
		}
		if kefuID, ok := c.Get("kefu_id"); ok {
			fields = append(fields, zap.Any("kefu_id", kefuID))
		}
		if entID, ok := c.Get("ent_id"); ok {
			fields = append(fields, zap.Any("ent_id", entID))
		}
		if errorsText := c.Errors.String(); errorsText != "" {
			fields = append(fields, zap.String("errors", errorsText))
		}

		message := c.Request.Method + " " + path
		switch {
		case status >= http.StatusInternalServerError:
			zap.L().Error(message, fields...)
		case status >= http.StatusBadRequest:
			zap.L().Warn(message, fields...)
		case duration >= 2*time.Second:
			fields = append(fields, zap.Bool("slow", true))
			zap.L().Warn(message, fields...)
		case path == "/healthz" || path == "/readyz":
			zap.L().Debug(message, fields...)
		default:
			zap.L().Info(message, fields...)
		}
	}
}

// GinRecovery converts panics into a 500 response and records a structured
// entry without dumping authorization headers or other request secrets.
func GinRecovery(includeStack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			requestID, _ := c.Get(RequestIDKey)
			fields := []zap.Field{
				zap.String("component", "recovery"),
				zap.Any("error", recovered),
				zap.Any("request_id", requestID),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("query", sanitizeQuery(c.Request.URL.Query())),
				zap.String("ip", c.ClientIP()),
			}
			if includeStack {
				fields = append(fields, zap.ByteString("stack", debug.Stack()))
			}

			if isBrokenConnection(recovered) {
				zap.L().Warn("client connection closed", fields...)
				if err, ok := recovered.(error); ok {
					_ = c.Error(err)
				}
				c.Abort()
				return
			}

			zap.L().Error("panic recovered", fields...)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":       http.StatusInternalServerError,
				"msg":        "服务器内部错误",
				"request_id": requestID,
			})
		}()
		c.Next()
	}
}

func resolveRequestID(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if len(candidate) >= 8 && len(candidate) <= 128 {
		valid := true
		for _, char := range candidate {
			if (char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == '_' || char == '.' {
				continue
			}
			valid = false
			break
		}
		if valid {
			return candidate
		}
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err == nil {
		return hex.EncodeToString(random)
	}
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func sanitizeQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	clean := make(url.Values, len(values))
	for key, value := range values {
		if isSensitiveKey(key) {
			clean.Set(key, "[REDACTED]")
			continue
		}
		clean[key] = append([]string(nil), value...)
	}
	return clean.Encode()
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", ""), "_", ""))
	switch normalized {
	case "token", "authorization", "password", "passwd", "pwd", "secret",
		"apikey", "accesskey", "accesstoken", "refreshtoken", "code":
		return true
	default:
		return strings.Contains(normalized, "password") ||
			strings.Contains(normalized, "secret") ||
			strings.HasSuffix(normalized, "token")
	}
}

func isBrokenConnection(recovered interface{}) bool {
	var networkError *net.OpError
	err, ok := recovered.(error)
	if !ok || !errors.As(err, &networkError) {
		return false
	}
	text := strings.ToLower(networkError.Error())
	return strings.Contains(text, "broken pipe") ||
		strings.Contains(text, "connection reset by peer")
}
