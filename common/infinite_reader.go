package common

type InfiniteReader struct{}

func (is InfiniteReader) Read(b []byte) (int, error) {
	return len(b), nil
}
