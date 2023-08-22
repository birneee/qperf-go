package qlog

import (
	"bufio"
	"io"
	"os"
	"path"
	"qperf-go/common/utils"
)

// NewStdoutQlogWriter writes everything to a stdout
func NewStdoutQlogWriter(config *Config) Writer {
	return NewQlogWriter(&NotClosingWriteCloser{os.Stdout}, config)
}

// NewFileQlogWriter writes everything to a single file
func NewFileQlogWriter(filepath string, config *Config) Writer {
	err := os.MkdirAll(path.Dir(filepath), 0700)
	if err != nil {
		panic(err)
	}
	f, err := os.Create(filepath)
	if err != nil {
		panic(err)
	}
	w := utils.NewBufferedWriteCloser(bufio.NewWriter(f), f)
	return NewQlogWriter(w, config)
}

type NotClosingWriteCloser struct {
	inner io.WriteCloser
}

func (w *NotClosingWriteCloser) Write(p []byte) (n int, err error) {
	return w.inner.Write(p)
}

func (w *NotClosingWriteCloser) Close() error {
	// ignore
	return nil
}
