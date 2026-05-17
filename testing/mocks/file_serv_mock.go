package mocks

import (
	"backend/models/domains"
	"mime/multipart"

	"github.com/stretchr/testify/mock"
)

type FileServMock struct {
	mock.Mock
}

func (m *FileServMock) UploadFiles(files []*multipart.FileHeader) ([]domains.File, error) {
	args := m.Called(files)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domains.File), args.Error(1)
}

func (m *FileServMock) DeleteFile(fileURL string) error {
	args := m.Called(fileURL)
	return args.Error(0)
}
