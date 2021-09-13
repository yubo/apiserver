package native

import (
	"os"
	"path/filepath"

	"github.com/yubo/golib/stream"
	"github.com/yubo/golib/util"
)

type RecorderProvider interface {
	Open(path string) (stream.Recorder, error)
}

func NewFileRecorderProvider(dir string) (RecorderProvider, error) {
	return &fileRecorderProvider{prefixPath: dir}, nil
}

type fileRecorderProvider struct {
	prefixPath string
}

func (p *fileRecorderProvider) Open(path string) (stream.Recorder, error) {
	filePath := filepath.Join(p.prefixPath, path)
	fileDir := filepath.Dir(filePath)
	if !util.IsDir(fileDir) {
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			return nil, err
		}
	}

	fd, err := os.OpenFile(filePath,
		os.O_RDWR|os.O_CREATE|os.O_EXCL|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	return stream.NewRecorder(fd)
}
