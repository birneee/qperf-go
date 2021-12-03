package common

import (
	"crypto/x509"
	"io/ioutil"
)

func NewCertPoolWithCert(tlsCertFile string) *x509.CertPool {
	certPool := x509.NewCertPool()
	caCertRaw, err := ioutil.ReadFile(tlsCertFile)
	if err != nil {
		panic(err)
	}

	ok := certPool.AppendCertsFromPEM(caCertRaw)
	if !ok {
		panic("failed to add certificate to pool")
	}
	return certPool
}
