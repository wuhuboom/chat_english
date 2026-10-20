package controller

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go-fly-muti/tools"
)

func allowUploadRequest(c *gin.Context, maxBytes int64) bool {
	if !tools.SecurityLimitFreqSingle("upload:"+c.ClientIP(), 20, 60) {
		c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "msg": "上传过于频繁，请稍后重试"})
		return false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes+1024*1024)
	return true
}

const (
	maxImageUpload = 10 * 1024 * 1024
	maxFileUpload  = 32 * 1024 * 1024
	maxAudioUpload = 20 * 1024 * 1024
)

var allowedAttachmentExtensions = map[string]bool{
	".pdf": true, ".txt": true, ".zip": true, ".rar": true, ".7z": true,
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".mp4": true,
}

func validateUpload(header *multipart.FileHeader, kind string) (string, error) {
	if header == nil || header.Size <= 0 {
		return "", errors.New("文件为空")
	}
	ext := strings.ToLower(filepath.Ext(filepath.Base(header.Filename)))
	var maxSize int64
	switch kind {
	case "image":
		maxSize = maxImageUpload
		if !map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}[ext] {
			return "", errors.New("只允许 jpg、jpeg、png、gif、webp 图片")
		}
	case "attachment":
		maxSize = maxFileUpload
		if !allowedAttachmentExtensions[ext] {
			return "", errors.New("该附件类型不允许上传")
		}
	case "audio":
		maxSize = maxAudioUpload
		ext = ".wav"
	default:
		return "", errors.New("未知上传类型")
	}
	if header.Size > maxSize {
		return "", errors.New("文件大小超过限制")
	}

	f, err := header.Open()
	if err != nil {
		return "", errors.New("无法读取上传文件")
	}
	defer f.Close()
	buf := make([]byte, 4096)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", errors.New("无法读取上传文件")
	}
	if !validUploadSignature(ext, buf[:n]) {
		return "", errors.New("文件内容与扩展名不符")
	}
	return ext, nil
}

func validUploadSignature(ext string, data []byte) bool {
	has := func(prefix []byte) bool { return bytes.HasPrefix(data, prefix) }
	switch ext {
	case ".jpg", ".jpeg":
		return has([]byte{0xff, 0xd8, 0xff})
	case ".png":
		return has([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
	case ".gif":
		return has([]byte("GIF87a")) || has([]byte("GIF89a"))
	case ".webp":
		return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	case ".pdf":
		return has([]byte("%PDF-"))
	case ".zip", ".docx", ".xlsx", ".pptx":
		return has([]byte{'P', 'K', 0x03, 0x04}) || has([]byte{'P', 'K', 0x05, 0x06})
	case ".rar":
		return has([]byte("Rar!\x1a\x07"))
	case ".7z":
		return has([]byte{'7', 'z', 0xbc, 0xaf, 0x27, 0x1c})
	case ".doc", ".xls", ".ppt":
		return has([]byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1})
	case ".mp4":
		return len(data) >= 12 && string(data[4:8]) == "ftyp"
	case ".wav":
		return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE"
	case ".txt":
		trimmed := bytes.TrimSpace(data)
		lower := strings.ToLower(string(trimmed))
		return utf8.Valid(data) && !strings.HasPrefix(lower, "<html") && !strings.HasPrefix(lower, "<!doctype") && !strings.Contains(lower, "<script")
	default:
		return false
	}
}
