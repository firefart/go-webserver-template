package http

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func getCertificateChain(certPath string) (*x509.CertPool, error) {
	rootCAs, err := x509.SystemCertPool()
	if rootCAs == nil || err != nil {
		rootCAs = x509.NewCertPool()
	}

	if err := filepath.WalkDir(certPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}
		if d.IsDir() {
			// ignore directories
			return nil
		}

		ext := filepath.Ext(d.Name())
		switch ext {
		case ".crt":
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read cert file %s: %w", d.Name(), err)
			}
			cert, err := x509.ParseCertificate(content)
			if err != nil {
				return fmt.Errorf("failed to parse crt file %s: %w", d.Name(), err)
			}
			// Encode to PEM
			pemBytes := pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: cert.Raw,
			})
			// Append our cert to the system pool
			if ok := rootCAs.AppendCertsFromPEM(pemBytes); !ok {
				return fmt.Errorf("failed to append crt from %s", d.Name())
			}
		case ".pem":
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read cert file %s: %w", d.Name(), err)
			}
			// Append our cert to the system pool
			if ok := rootCAs.AppendCertsFromPEM(content); !ok {
				return fmt.Errorf("failed to append cert from %s", d.Name())
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("error walking the path %s: %w", certPath, err)
	}

	return rootCAs, nil
}
