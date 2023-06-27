package utils

import "io"

type InfiniteReader struct {
}

var _ io.Reader = InfiniteReader{}

func (i InfiniteReader) Read(p []byte) (n int, err error) {
	return len(p), nil
}

type funcToWriterStruct struct {
	writeFunc func(p []byte) (n int, err error)
}

func (f funcToWriterStruct) Write(p []byte) (n int, err error) {
	return f.writeFunc(p)
}

func FuncToWriter(writeFunc func(p []byte) (n int, err error)) io.Writer {
	return funcToWriterStruct{
		writeFunc: writeFunc,
	}
}
