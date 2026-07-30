package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultQueryLimit = 100
	maxQueryLimit     = 500
	maxTailBytes      = 8 * 1024 * 1024
)

type LogEntry struct {
	Time          string `json:"time"`
	Level         string `json:"level"`
	Message       string `json:"message"`
	Caller        string `json:"caller,omitempty"`
	Component     string `json:"component,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	Method        string `json:"method,omitempty"`
	Path          string `json:"path,omitempty"`
	Status        int    `json:"status,omitempty"`
	Duration      string `json:"duration,omitempty"`
	IP            string `json:"ip,omitempty"`
	KefuID        string `json:"kefu_id,omitempty"`
	EntID         string `json:"ent_id,omitempty"`
	ResponseBytes int    `json:"response_bytes,omitempty"`
	Raw           string `json:"raw"`
}

type LogQuery struct {
	Level     string
	Keyword   string
	RequestID string
	Limit     int
}

type LogQueryResult struct {
	Entries      []LogEntry     `json:"entries"`
	Counts       map[string]int `json:"counts"`
	File         string         `json:"file"`
	FileSize     int64          `json:"file_size"`
	ModifiedAt   string         `json:"modified_at"`
	ScannedLines int            `json:"scanned_lines"`
	Truncated    bool           `json:"truncated"`
}

func Query(query LogQuery) (LogQueryResult, error) {
	result := LogQueryResult{
		Entries: make([]LogEntry, 0),
		Counts: map[string]int{
			"debug": 0,
			"info":  0,
			"warn":  0,
			"error": 0,
		},
		File: FilePath(),
	}
	query.Level = strings.ToLower(strings.TrimSpace(query.Level))
	query.Keyword = strings.ToLower(strings.TrimSpace(query.Keyword))
	query.RequestID = strings.TrimSpace(query.RequestID)
	if query.Limit <= 0 {
		query.Limit = defaultQueryLimit
	}
	if query.Limit > maxQueryLimit {
		query.Limit = maxQueryLimit
	}

	info, err := os.Stat(result.File)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, fmt.Errorf("stat log file: %w", err)
	}
	result.FileSize = info.Size()
	result.ModifiedAt = info.ModTime().Format(time.RFC3339)

	lines, truncated, err := tailLines(result.File, maxTailBytes)
	if err != nil {
		return result, err
	}
	result.Truncated = truncated
	result.ScannedLines = len(lines)

	for index := len(lines) - 1; index >= 0; index-- {
		entry := parseEntry(lines[index])
		result.Counts[entry.Level]++
		if query.Level != "" && query.Level != "all" && entry.Level != query.Level {
			continue
		}
		if query.RequestID != "" && entry.RequestID != query.RequestID {
			continue
		}
		if query.Keyword != "" && !strings.Contains(strings.ToLower(entry.Raw), query.Keyword) {
			continue
		}
		result.Entries = append(result.Entries, entry)
		if len(result.Entries) >= query.Limit {
			break
		}
	}
	return result, nil
}

func tailLines(filename string, maxBytes int64) ([]string, bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, false, fmt.Errorf("open log file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("stat log file: %w", err)
	}
	start := info.Size() - maxBytes
	truncated := start > 0
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, 0); err != nil {
		return nil, false, fmt.Errorf("seek log file: %w", err)
	}
	data := make([]byte, info.Size()-start)
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, false, fmt.Errorf("read log file: %w", err)
	}
	if truncated {
		if separator := bytes.IndexByte(data, '\n'); separator >= 0 {
			data = data[separator+1:]
		}
	}
	rawLines := strings.Split(strings.TrimSpace(string(data)), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, truncated, nil
}

func parseEntry(raw string) LogEntry {
	entry := LogEntry{Level: inferLegacyLevel(raw), Message: raw, Raw: raw}
	var values map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return entry
	}
	entry.Time = stringValue(values["time"])
	entry.Level = normalizeLevel(stringValue(values["level"]))
	entry.Message = firstString(values, "message", "msg")
	entry.Caller = stringValue(values["caller"])
	entry.Component = stringValue(values["component"])
	entry.RequestID = stringValue(values["request_id"])
	entry.Method = stringValue(values["method"])
	entry.Path = stringValue(values["path"])
	entry.Status = intValue(values["status"])
	entry.Duration = durationValue(values["duration"])
	entry.IP = stringValue(values["ip"])
	entry.KefuID = stringValue(values["kefu_id"])
	entry.EntID = stringValue(values["ent_id"])
	entry.ResponseBytes = intValue(values["response_bytes"])
	return entry
}

func inferLegacyLevel(raw string) string {
	upper := strings.ToUpper(raw)
	switch {
	case strings.Contains(upper, "\tERROR\t") || strings.Contains(upper, "[ERROR]"):
		return "error"
	case strings.Contains(upper, "\tWARN\t") || strings.Contains(upper, "[WARN]"):
		return "warn"
	case strings.Contains(upper, "\tDEBUG\t") || strings.Contains(upper, "[DEBUG]"):
		return "debug"
	default:
		return "info"
	}
}

func normalizeLevel(level string) string {
	switch strings.ToLower(level) {
	case "debug":
		return "debug"
	case "warn", "warning":
		return "warn"
	case "error", "fatal", "panic", "dpanic":
		return "error"
	default:
		return "info"
	}
}

func firstString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

func intValue(value interface{}) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		parsed, _ := strconv.Atoi(typed)
		return parsed
	default:
		return 0
	}
}

func durationValue(value interface{}) string {
	switch typed := value.(type) {
	case float64:
		return fmt.Sprintf("%.2fms", typed)
	case string:
		return typed
	default:
		return ""
	}
}
