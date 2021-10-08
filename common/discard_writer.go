package common

type DiscardWriter struct{}

func (dw DiscardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
