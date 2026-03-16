package services

import (
	"backend/models/domains"
	"mime/multipart"
)

type FileServ interface {
	UploadFiles(files []*multipart.FileHeader) ([]domains.File, error)
	DeleteFile(fileURL string) error
}
