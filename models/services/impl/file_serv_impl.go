package impl

import (
	"backend/models/domains"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	MaxImageSize = 50 << 20
	MaxDocSize   = 10 << 20
	UploadDir    = "assets"
)

type FileServImpl struct{}

func NewFileServImpl() *FileServImpl {
	return &FileServImpl{}
}

func (serv *FileServImpl) UploadFiles(files []*multipart.FileHeader) ([]domains.File, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	if err := os.MkdirAll(UploadDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	allowedExts := map[string]int64{
		".jpg":  MaxImageSize,
		".jpeg": MaxImageSize,
		".png":  MaxImageSize,
		".webp": MaxImageSize,
		".pdf":  MaxDocSize,
		".doc":  MaxDocSize,
		".docx": MaxDocSize,
	}

	var uploaded []domains.File

	for _, fileHeader := range files {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		maxSize, allowed := allowedExts[ext]
		if !allowed {
			return nil, fmt.Errorf("file type not allowed: %s", ext)
		}
		if fileHeader.Size > maxSize {
			return nil, fmt.Errorf("file %s too large (max %d MB)", fileHeader.Filename, maxSize>>20)
		}

		baseName := strings.TrimSuffix(fileHeader.Filename, ext)
		safeBaseName := sanitizeFileName(baseName)
		fileName := safeBaseName + ext
		dstPath := filepath.Join(UploadDir, fileName)

		for counter := 1; ; counter++ {
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				break
			}
			fileName = fmt.Sprintf("%s(%d)%s", safeBaseName, counter, ext)
			dstPath = filepath.Join(UploadDir, fileName)
		}

		src, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", fileHeader.Filename, err)
		}

		dst, err := os.Create(dstPath)
		if err != nil {
			if errSrc := src.Close(); errSrc != nil {
				return nil, fmt.Errorf("failed to close source file %s: %w", fileName, errSrc)
			}
			return nil, fmt.Errorf("failed to create file %s: %w", fileName, err)
		}

		_, errCopy := io.Copy(dst, src)
		errDst := dst.Close()
		errSrc := src.Close()

		if errCopy != nil {
			if errRemove := os.Remove(dstPath); errRemove != nil {
				return nil, fmt.Errorf("failed to remove file %s: %w", fileName, errRemove)
			}
			return nil, fmt.Errorf("failed to save file %s: %w", fileName, errCopy)
		}
		if errDst != nil {
			return nil, fmt.Errorf("failed to close destination file %s: %w", fileName, errDst)
		}
		if errSrc != nil {
			return nil, fmt.Errorf("failed to close source file %s: %w", fileName, errSrc)
		}

		uploaded = append(uploaded, domains.File{
			FileName: fileName,
			FileURL:  fmt.Sprintf("/assets/%s", fileName),
		})
	}

	return uploaded, nil
}

func (serv *FileServImpl) DeleteFile(fileURL string) error {
	if fileURL == "" {
		return fmt.Errorf("file URL is required")
	}

	path := strings.TrimPrefix(fileURL, "/")
	if !strings.HasPrefix(path, UploadDir+"/") {
		return fmt.Errorf("invalid file path")
	}

	fullPath := filepath.Join(".", path)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", fileURL)
		}
		return fmt.Errorf("failed to delete file %s: %w", fileURL, err)
	}

	return nil
}

func sanitizeFileName(name string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\- ]+`)
	safe := reg.ReplaceAllString(name, "")
	safe = strings.ReplaceAll(safe, " ", "_")
	if safe == "" {
		safe = "file"
	}
	if len(safe) > 100 {
		safe = safe[:100]
	}
	return safe
}
