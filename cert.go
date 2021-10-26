package vproxy

import (
	"log"
	"os"
	"path/filepath"

	"github.com/jittering/truststore"
	"github.com/mitchellh/go-homedir"
)

var ts *truststore.MkcertLib

func InitTrustStore() error {
	var err error
	ts, err = truststore.NewLib()
	return err
}

func CARootPath() string {
	if cp := os.Getenv("CAROOT_PATH"); cp != "" {
		// override from env
		return cp
	}

	return truststore.GetCAROOT()
}

func CertPath() string {
	if cp := os.Getenv("CERT_PATH"); cp != "" {
		// override from env
		return cp
	}

	// default to user homedir
	d, err := homedir.Dir()
	if err != nil {
		log.Fatalf("failed to locate homedir: %s", err)
	}
	return filepath.Join(d, ".vproxy")
}

// MakeCert for the give hostname, if it doesn't already exist.
func MakeCert(host string) (certFile string, keyFile string, err error) {
	cp := CertPath()
	err = os.MkdirAll(cp, 0755)
	if err != nil {
		return "", "", err
	}

	cert, err := ts.CertFile([]string{host}, cp)
	if err != nil {
		return "", "", err
	}
	if exists(cert.CertFile) && exists(cert.KeyFile) {
		// nothing to do
		return certFile, keyFile, nil
	}

	cert, err = ts.MakeCert([]string{host}, cp)
	if err != nil {
		return "", "", err
	}
	return cert.CertFile, cert.KeyFile, nil
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}
