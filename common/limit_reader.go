package common

import "io"

// LimitReader returns a Reader that reads from r.
// copy of go/io.go LimitReader but using uint64 instead of int64
// but stops with EOF after n bytes.
// The underlying implementation is a *LimitedReader.
func LimitReader(r io.Reader, n uint64) io.Reader { return &LimitedReader{r, n} }

// A LimitedReader reads from R but limits the amount of
// data returned to just N bytes. Each call to Read
// updates N to reflect the new amount remaining.
// Read returns EOF when N <= 0 or when the underlying R returns EOF.
type LimitedReader struct {
	R io.Reader // underlying reader
	N uint64    // max bytes remaining
}

func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, io.EOF
	}
	if uint64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= uint64(n)
	return
}
