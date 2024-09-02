package common

import "io"

type countingReader struct {
	reader io.Reader
	count  func(int)
}

func NewCountingReader(reader io.Reader, count func(int)) io.Reader {
	return &countingReader{
		reader: reader,
		count:  count,
	}
}

func (c countingReader) Read(p []byte) (n int, err error) {
	n, err = c.reader.Read(p)
	c.count(n)
	return
}
