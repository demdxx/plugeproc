package executor

import (
	"errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
)

var ErrCantCreateTmpFile = errors.New("invalid tmp file creation")

type tmpFile struct {
	filepath string
	f        *os.File
}

func tempFrom(in io.Reader) (*tmpFile, error) {
	filepath, err := tempFilepath()
	if err != nil {
		return nil, err
	}
	if in != nil {
		if err := writeReaderFile(filepath, in); err != nil {
			return nil, err
		}
	}
	return tempOpen(filepath)
}

func tempOpen(filepath string) (*tmpFile, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	return &tmpFile{filepath: filepath, f: f}, nil
}

func (f *tmpFile) Read(b []byte) (int, error) {
	return f.f.Read(b)
}

func (f *tmpFile) ReadAt(b []byte, off int64) (int, error) {
	return f.f.ReadAt(b, off)
}

func (f *tmpFile) Write(b []byte) (int, error) {
	return f.f.Write(b)
}

func (f *tmpFile) Close() error {
	if err := f.f.Close(); err != nil {
		return err
	}
	if f.filepath != "" {
		if err := os.Remove(f.filepath); err != nil {
			return err
		}
		f.filepath = ""
	}
	return nil
}

func tempFilepath() (string, error) {
	dir := os.TempDir()
	pattern := "eplug-obj-"
	for i := 0; i < 10; i++ {
		path := filepath.Join(dir, pattern+strconv.Itoa(rand.Int()))
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return path, nil
		}
	}
	return "", ErrCantCreateTmpFile
}

func writeReaderFile(path string, input io.Reader) error {
	inputFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = inputFile.Close()
	}()
	_, err = io.Copy(inputFile, input)
	return err
}

func tempFileCreate(input io.Reader) (string, error) {
	fpath, err := tempFilepath()
	if err != nil {
		return "", err
	}
	return fpath, writeReaderFile(fpath, input)
}
