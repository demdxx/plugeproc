package driver

import (
	"errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
)

var errCantCreateTmpFile = errors.New("cannot create temp file")

type tmpFile struct {
	path string
	f    *os.File
}

func tempFrom(in io.Reader) (*tmpFile, error) {
	path, err := tempFilepath()
	if err != nil {
		return nil, err
	}
	if in != nil {
		if err := writeReaderFile(path, in); err != nil {
			return nil, err
		}
	}
	return tempOpen(path)
}

func tempOpen(path string) (*tmpFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &tmpFile{path: path, f: f}, nil
}

func (f *tmpFile) Read(b []byte) (int, error)             { return f.f.Read(b) }
func (f *tmpFile) ReadAt(b []byte, off int64) (int, error) { return f.f.ReadAt(b, off) }
func (f *tmpFile) Write(b []byte) (int, error)             { return f.f.Write(b) }

func (f *tmpFile) Close() error {
	if err := f.f.Close(); err != nil {
		return err
	}
	if f.path != "" {
		if err := os.Remove(f.path); err != nil {
			return err
		}
		f.path = ""
	}
	return nil
}

func tempFilepath() (string, error) {
	dir := os.TempDir()
	for i := 0; i < 10; i++ {
		path := filepath.Join(dir, "eplug-obj-"+strconv.Itoa(rand.Int()))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		}
	}
	return "", errCantCreateTmpFile
}

func writeReaderFile(path string, input io.Reader) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, input)
	return err
}

func tempFileCreate(input io.Reader) (string, error) {
	path, err := tempFilepath()
	if err != nil {
		return "", err
	}
	return path, writeReaderFile(path, input)
}
