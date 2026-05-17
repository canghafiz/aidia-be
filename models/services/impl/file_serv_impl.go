package impl

import (
	"backend/models/domains"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	MaxImageSize = 50 << 20
	MaxDocSize   = 10 << 20
	UploadDir    = "assets"
)

var allowedExts = map[string]int64{
	".jpg":  MaxImageSize,
	".jpeg": MaxImageSize,
	".png":  MaxImageSize,
	".webp": MaxImageSize,
	".pdf":  MaxDocSize,
	".doc":  MaxDocSize,
	".docx": MaxDocSize,
}

type FileServImpl struct {
	s3Client   *s3.Client
	bucket     string
	publicBase string
}

func NewFileServImpl() *FileServImpl {
	endpoint   := os.Getenv("US3_ENDPOINT")
	bucket     := os.Getenv("US3_BUCKET")
	accessKey  := os.Getenv("US3_ACCESS_KEY")
	secretKey  := os.Getenv("US3_SECRET_KEY")
	publicBase := os.Getenv("US3_PUBLIC_URL")

	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return &FileServImpl{}
	}

	if publicBase == "" {
		publicBase = fmt.Sprintf("https://%s.%s", bucket, endpoint)
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return &FileServImpl{}
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s", endpoint))
		o.UsePathStyle = true
	})

	return &FileServImpl{
		s3Client:   client,
		bucket:     bucket,
		publicBase: strings.TrimRight(publicBase, "/"),
	}
}

func (serv *FileServImpl) UploadFiles(files []*multipart.FileHeader) ([]domains.File, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}
	if serv.s3Client != nil {
		return serv.uploadToS3(files)
	}
	return serv.uploadToLocal(files)
}

func (serv *FileServImpl) uploadToS3(files []*multipart.FileHeader) ([]domains.File, error) {
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
		fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), sanitizeFileName(baseName), ext)

		src, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", fileHeader.Filename, err)
		}
		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", fileHeader.Filename, err)
		}

		_, err = serv.s3Client.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket:      aws.String(serv.bucket),
			Key:         aws.String(fileName),
			Body:        bytes.NewReader(data),
			ContentType: aws.String(extMimeType(ext)),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upload %s to US3: %w", fileName, err)
		}

		uploaded = append(uploaded, domains.File{
			FileName: fileName,
			FileURL:  fmt.Sprintf("%s/%s", serv.publicBase, fileName),
		})
	}

	return uploaded, nil
}

func (serv *FileServImpl) uploadToLocal(files []*multipart.FileHeader) ([]domains.File, error) {
	if err := os.MkdirAll(UploadDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
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
		safeBase := sanitizeFileName(baseName)
		fileName := safeBase + ext
		dstPath := filepath.Join(UploadDir, fileName)
		for counter := 1; ; counter++ {
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				break
			}
			fileName = fmt.Sprintf("%s(%d)%s", safeBase, counter, ext)
			dstPath = filepath.Join(UploadDir, fileName)
		}

		src, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", fileHeader.Filename, err)
		}
		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			return nil, fmt.Errorf("failed to create %s: %w", fileName, err)
		}
		_, errCopy := io.Copy(dst, src)
		errDst := dst.Close()
		errSrc := src.Close()
		if errCopy != nil {
			os.Remove(dstPath)
			return nil, fmt.Errorf("failed to save %s: %w", fileName, errCopy)
		}
		if errDst != nil {
			return nil, fmt.Errorf("failed to close destination %s: %w", fileName, errDst)
		}
		if errSrc != nil {
			return nil, fmt.Errorf("failed to close source %s: %w", fileName, errSrc)
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

	if serv.s3Client != nil {
		key := strings.TrimPrefix(fileURL, serv.publicBase+"/")
		_, err := serv.s3Client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket: aws.String(serv.bucket),
			Key:    aws.String(key),
		})
		return err
	}

	path := strings.TrimPrefix(fileURL, "/")
	if !strings.HasPrefix(path, UploadDir+"/") {
		return fmt.Errorf("invalid file path")
	}
	if err := os.Remove(filepath.Join(".", path)); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", fileURL)
		}
		return fmt.Errorf("failed to delete %s: %w", fileURL, err)
	}
	return nil
}

func extMimeType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
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
