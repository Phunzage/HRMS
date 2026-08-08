package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 上传限制：单文件最大 5MB
const maxUploadSize = 5 << 20

// allowedImageExts 允许上传的图片扩展名 → 对应 MIME 类型（扩展名 + Content-Type 双重校验）
var allowedImageExts = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
}

// 上传相关业务错误
var (
	ErrFileTooLarge     = errors.New("文件大小不能超过 5MB")
	ErrFileTypeNotAllow = errors.New("仅支持 jpg/png/webp/gif 图片")
)

// UploadService 文件上传服务：校验图片类型与大小，保存到 uploads/YYYYMM/ 子目录
type UploadService struct {
	baseDir string // 上传根目录（相对后端运行目录，如 ./uploads）
}

// NewUploadService 创建上传服务
func NewUploadService(baseDir string) *UploadService {
	return &UploadService{baseDir: baseDir}
}

// UploadResult 上传成功返回的可公开访问相对路径
type UploadResult struct {
	URL string `json:"url"`
}

// SaveImage 校验并保存上传的图片，返回 /uploads/YYYYMM/<uuid>.ext 形式的相对路径
func (s *UploadService) SaveImage(fileHeader *multipart.FileHeader) (*UploadResult, error) {
	// 1. 大小校验
	if fileHeader.Size > maxUploadSize {
		return nil, ErrFileTooLarge
	}
	// 2. 扩展名 + Content-Type 双重校验：扩展名必须在白名单内；
	// Content-Type 需与扩展名一致；部分客户端（如 Go/curl 通用上传）会发 application/octet-stream，
	// 此时以扩展名为准（仍限制在图片白名单内）
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	contentType := fileHeader.Header.Get("Content-Type")
	expected, ok := allowedImageExts[ext]
	if !ok || (contentType != expected && contentType != "application/octet-stream") {
		return nil, ErrFileTypeNotAllow
	}
	// 3. 生成随机文件名并写入 YYYYMM 子目录
	randName, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	subDir := time.Now().Format("200601")
	dir := filepath.Join(s.baseDir, subDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	filename := randName + ext
	dst := filepath.Join(dir, filename)

	src, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		return nil, err
	}
	return &UploadResult{URL: "/uploads/" + subDir + "/" + filename}, nil
}

// randomHex 生成 n 字节的随机十六进制字符串（等价于 2n 字符的随机文件名）
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
