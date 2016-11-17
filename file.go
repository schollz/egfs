package egfs

import (
	"bytes"
	"os"
	"time"
)

type file struct {
	content *bytes.Buffer // actual file contents
	name    string        // actual name of file
	modTime time.Time     // time file was last modified
}

func (f *file) Close() error {
	return nil
}
func (f *file) Stat() (os.FileInfo, error) {
	return &fileInfo{f}, nil
}
func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	return []os.FileInfo{&fileInfo{f}}, nil
}
func (f *file) Read(b []byte) (int, error) {
	return f.content.Read(b)
}
func (f *file) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

type fileInfo struct {
	file *file
}

// Implements os.FileInfo
func (s *fileInfo) Name() string       { return s.file.name }
func (s *fileInfo) Size() int64        { return int64(s.file.content.Len()) }
func (s *fileInfo) Mode() os.FileMode  { return os.ModeTemporary }
func (s *fileInfo) ModTime() time.Time { return s.file.modTime }
func (s *fileInfo) IsDir() bool        { return false }
func (s *fileInfo) Sys() interface{}   { return nil }
