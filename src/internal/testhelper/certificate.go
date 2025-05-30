package testhelper

import (
	"log"
	"os"

	"code.cloudfoundry.org/tlsconfig/certtest"
)

type TestCerts struct {
	ca *certtest.Authority

	caFile       string
	certKeyPairs map[string]certKeyPair
}

type certKeyPair struct {
	certFile string
	keyFile  string
}

func (tc *TestCerts) CA() string {
	return tc.caFile
}

func (tc *TestCerts) Cert(commonName string) string {
	return tc.keyPair(commonName).certFile
}

func (tc *TestCerts) Key(commonName string) string {
	return tc.keyPair(commonName).keyFile
}

func (tc *TestCerts) keyPair(commonName string) certKeyPair {
	keyPair, ok := tc.certKeyPairs[commonName]
	if !ok {
		keyPair = tc.generateCertKeyPair(commonName)
		tc.certKeyPairs[commonName] = keyPair
	}

	return keyPair
}

func GenerateCerts(caName string) *TestCerts {
	ca, caFile := generateCA(caName)

	return &TestCerts{
		ca:           ca,
		caFile:       caFile,
		certKeyPairs: map[string]certKeyPair{},
	}
}

func generateCA(caName string) (*certtest.Authority, string) {
	ca, err := certtest.BuildCA(caName)
	if err != nil {
		log.Fatal(err)
	}

	caBytes, err := ca.CertificatePEM()
	if err != nil {
		log.Fatal(err)
	}

	fileName := tmpFile(caName+".crt", caBytes)

	return ca, fileName
}

func tmpFile(prefix string, caBytes []byte) string {
	file, err := os.CreateTemp("", prefix)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close() //nolint:errcheck

	_, err = file.Write(caBytes)
	if err != nil {
		log.Fatal(err)
	}

	return file.Name()
}

func (tc *TestCerts) generateCertKeyPair(commonName string) certKeyPair {
	cert, err := tc.ca.BuildSignedCertificate(commonName, certtest.WithDomains(commonName))
	if err != nil {
		log.Fatal(err)
	}

	certBytes, keyBytes, err := cert.CertificatePEMAndPrivateKey()
	if err != nil {
		log.Fatal(err)
	}

	certFile := tmpFile(commonName+".crt", certBytes)
	keyFile := tmpFile(commonName+".key", keyBytes)

	return certKeyPair{
		certFile: certFile,
		keyFile:  keyFile,
	}
}
