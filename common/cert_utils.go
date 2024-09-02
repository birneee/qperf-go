package common

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

func NewCertPoolFromFiles(files ...string) *x509.CertPool {
	certPool := x509.NewCertPool()
	for _, file := range files {
		caCertRaw, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}

		ok := certPool.AppendCertsFromPEM(caCertRaw)
		if !ok {
			panic("failed to add certificate to pool")
		}
	}
	return certPool
}

func GenerateCert() tls.Certificate {
	return GenerateCertFor([]string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("127.0.0.2")})
}

func GenerateCertFor(dnsNames []string, ipAddresses []net.IP) tls.Certificate {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, 10),
		DNSNames:     dnsNames,
		IPAddresses:  ipAddresses,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return tlsCert
}
