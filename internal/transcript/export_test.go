package transcript

import (
	"io"
	"io/fs"
	"os"
)

// Counted reads a transcript through a wrapper that counts every byte read, and returns the total.
func Counted(path string) (info Info, total int64) {
	r := reader{open: func(p string) (source, error) {
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		return &countingFile{File: f, total: &total}, nil
	}}
	info = r.read(path)
	return info, total
}

type countingFile struct {
	*os.File
	total *int64
}

func (c *countingFile) Read(p []byte) (int, error) {
	n, err := c.File.Read(p)
	*c.total += int64(n)
	return n, err
}

func (c *countingFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := c.File.ReadAt(p, off)
	*c.total += int64(n)
	return n, err
}

func (c *countingFile) Stat() (fs.FileInfo, error) { return c.File.Stat() }

var _ io.ReaderAt = (*countingFile)(nil)
