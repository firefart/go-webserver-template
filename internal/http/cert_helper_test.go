package http

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

const (
	// Valid PEM-encoded self-signed certificate for testing
	// Generated with: openssl req -x509 -newkey rsa:2048 -nodes -keyout key.pem -out cert.pem -days 365 -subj "/CN=Test"
	validCertPEM = `-----BEGIN CERTIFICATE-----
MIICpDCCAYwCCQDU2jq7W8+2+jANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAlU
ZXN0IENlcnQwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjAUMRIwEAYD
VQQDDAlUZXN0IENlcnQwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC7
VJTUt9Us8cKjMzEfYyjiWA4R4/M2bS1+fWIcPm7m6ISS1hjMa3JCS1EmYDTZK736
gyKqkytt7KO1kZOJmJziuQqXp4ZLOyJVGXcV+E8aVTTxLLYP1FVW4jLRHXvYhqxJ
8ZSIIBBbMC8y9XGlUqMmUJBqJ7mHLi3w7XjqShFNWxPSTANgCfBWdLpOOJdEZS5w
9RkzQzEJJjJfhOzVLiIsPKTDEJJ0VmHMJKXVlL6zLpTDKYdX5RMz6FXQJ0VTQMQA
FhCpK7HTlhJqLHLQTdS6xf3Q2l1WPiCdE3uJ8QCSmTx7VqU+7bUUpQdZKcWIJVYJ
jKZvMkHaLpKJnKPE7+0xAgMBAAEwDQYJKoZIhvcNAQELBQADggEBADw3F3PJvDqF
cJNrYKmPVaGmZlYLhvg2DJGXA1BuJvXVUkJG8CKdFN+Dt6pWKY3vXQaGqPFWqTjV
WJPWVTm2gXKEGpdpEpAMqgVaLIkFgjUhVJVGW7pqYQxNJHqCQmYe1HRJQZYCkBVe
gg7o3FQlLlRvPdE4vHhHDZVZ1yNqWY4X+AUgK0eQSBgTLjK0xVJmDUdcL0YhH+qM
FDGLXUhBwHLO1R7sKLK7qL1gT5Y+mXQNqvYZKHGjWOGnLqJKQAFCNwBvWAA5QzPL
IvKEGgOcvJVHCbBvDzYTJcJnXVFqBpKPJxJGKVBzKPTKBPqTQqJYZFCNZvCYVsQN
7+VJcWJHUHI=
-----END CERTIFICATE-----`
)

// Helper function to convert PEM to DER format for .crt file testing
// nolint: unparam
func pemToDER(pemData string) ([]byte, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, x509.ErrUnsupportedAlgorithm
	}
	return block.Bytes, nil
}

func TestGetCertificateChain(t *testing.T) {
	t.Run("valid_certificates_in_directory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create .crt file with DER-encoded certificate
		derData, err := pemToDER(validCertPEM)
		if err != nil {
			t.Fatalf("failed to convert PEM to DER: %v", err)
		}
		certFile1 := filepath.Join(tempDir, "test1.crt")
		if err := os.WriteFile(certFile1, derData, 0o644); err != nil {
			t.Fatalf("failed to create test certificate: %v", err)
		}

		// Create .pem file with PEM-encoded certificate
		certFile2 := filepath.Join(tempDir, "test2.pem")
		if err := os.WriteFile(certFile2, []byte(validCertPEM), 0o644); err != nil {
			t.Fatalf("failed to create test certificate: %v", err)
		}

		pool, err := getCertificateChain(tempDir)
		if err != nil {
			t.Fatalf("getCertificateChain failed: %v", err)
		}

		if pool == nil {
			t.Fatal("expected non-nil certificate pool")
		}
	})

	t.Run("empty_directory", func(t *testing.T) {
		tempDir := t.TempDir()

		pool, err := getCertificateChain(tempDir)
		if err != nil {
			t.Fatalf("getCertificateChain failed: %v", err)
		}

		if pool == nil {
			t.Fatal("expected non-nil certificate pool")
		}
	})

	t.Run("non_existent_directory", func(t *testing.T) {
		nonExistentPath := filepath.Join(t.TempDir(), "does-not-exist")

		_, err := getCertificateChain(nonExistentPath)
		if err == nil {
			t.Fatal("expected error for non-existent directory")
		}
	})

	t.Run("invalid_certificate_content", func(t *testing.T) {
		tempDir := t.TempDir()

		// Invalid DER data for .crt file
		certFile := filepath.Join(tempDir, "invalid.crt")
		if err := os.WriteFile(certFile, []byte("invalid der data"), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := getCertificateChain(tempDir)
		if err == nil {
			t.Fatal("expected error for invalid certificate content")
		}
	})

	t.Run("mixed_file_types", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a valid DER certificate for .crt file
		derData, err := pemToDER(validCertPEM)
		if err != nil {
			t.Fatalf("failed to convert PEM to DER: %v", err)
		}
		certFile := filepath.Join(tempDir, "valid.crt")
		if err := os.WriteFile(certFile, derData, 0o644); err != nil {
			t.Fatalf("failed to create test certificate: %v", err)
		}

		// Create a file with non-certificate extension (should be ignored)
		txtFile := filepath.Join(tempDir, "readme.txt")
		if err := os.WriteFile(txtFile, []byte("This is a text file"), 0o644); err != nil {
			t.Fatalf("failed to create text file: %v", err)
		}

		// Create a file with no extension (should be ignored)
		noExtFile := filepath.Join(tempDir, "somefile")
		if err := os.WriteFile(noExtFile, []byte("Some content"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		pool, err := getCertificateChain(tempDir)
		if err != nil {
			t.Fatalf("getCertificateChain failed: %v", err)
		}

		if pool == nil {
			t.Fatal("expected non-nil certificate pool")
		}
	})

	t.Run("subdirectories_with_certificates", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create subdirectory
		subDir := filepath.Join(tempDir, "subdir")
		if err := os.Mkdir(subDir, 0o755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		// Create DER certificate in root
		derData, err := pemToDER(validCertPEM)
		if err != nil {
			t.Fatalf("failed to convert PEM to DER: %v", err)
		}
		rootCert := filepath.Join(tempDir, "root.crt")
		if err := os.WriteFile(rootCert, derData, 0o644); err != nil {
			t.Fatalf("failed to create root certificate: %v", err)
		}

		// Create PEM certificate in subdirectory
		subCert := filepath.Join(subDir, "sub.pem")
		if err := os.WriteFile(subCert, []byte(validCertPEM), 0o644); err != nil {
			t.Fatalf("failed to create subdirectory certificate: %v", err)
		}

		pool, err := getCertificateChain(tempDir)
		if err != nil {
			t.Fatalf("getCertificateChain failed: %v", err)
		}

		if pool == nil {
			t.Fatal("expected non-nil certificate pool")
		}
	})

	t.Run("unreadable_file", func(t *testing.T) {
		tempDir := t.TempDir()

		derData, err := pemToDER(validCertPEM)
		if err != nil {
			t.Fatalf("failed to convert PEM to DER: %v", err)
		}
		certFile := filepath.Join(tempDir, "test.crt")
		if err := os.WriteFile(certFile, derData, 0o000); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err = getCertificateChain(tempDir)
		if err == nil {
			t.Fatal("expected error for unreadable file")
		}
	})

	t.Run("both_crt_and_pem_extensions", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create .crt file with DER format
		derData, err := pemToDER(validCertPEM)
		if err != nil {
			t.Fatalf("failed to convert PEM to DER: %v", err)
		}
		crtFile := filepath.Join(tempDir, "cert.crt")
		if err := os.WriteFile(crtFile, derData, 0o644); err != nil {
			t.Fatalf("failed to create .crt file: %v", err)
		}

		// Create .pem file with PEM format
		pemFile := filepath.Join(tempDir, "cert.pem")
		if err := os.WriteFile(pemFile, []byte(validCertPEM), 0o644); err != nil {
			t.Fatalf("failed to create .pem file: %v", err)
		}

		pool, err := getCertificateChain(tempDir)
		if err != nil {
			t.Fatalf("getCertificateChain failed: %v", err)
		}

		if pool == nil {
			t.Fatal("expected non-nil certificate pool")
		}
	})

	t.Run("system_cert_pool_fallback", func(t *testing.T) {
		tempDir := t.TempDir()

		// Even with no custom certificates, should return system pool or new pool
		pool, err := getCertificateChain(tempDir)
		if err != nil {
			t.Fatalf("getCertificateChain failed: %v", err)
		}

		if pool == nil {
			t.Fatal("expected non-nil certificate pool")
		}
	})

	t.Run("empty_file", func(t *testing.T) {
		tempDir := t.TempDir()

		// Empty .crt file should fail DER parsing
		certFile := filepath.Join(tempDir, "empty.crt")
		if err := os.WriteFile(certFile, []byte(""), 0o644); err != nil {
			t.Fatalf("failed to create empty file: %v", err)
		}

		_, err := getCertificateChain(tempDir)
		if err == nil {
			t.Fatal("expected error for empty certificate file")
		}
	})

	t.Run("deeply_nested_subdirectories", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create deeply nested directory structure
		nestedPath := filepath.Join(tempDir, "level1", "level2", "level3")
		if err := os.MkdirAll(nestedPath, 0o755); err != nil {
			t.Fatalf("failed to create nested directories: %v", err)
		}

		// Create certificate in deeply nested directory
		certFile := filepath.Join(nestedPath, "nested.pem")
		if err := os.WriteFile(certFile, []byte(validCertPEM), 0o644); err != nil {
			t.Fatalf("failed to create nested certificate: %v", err)
		}

		pool, err := getCertificateChain(tempDir)
		if err != nil {
			t.Fatalf("getCertificateChain failed: %v", err)
		}

		if pool == nil {
			t.Fatal("expected non-nil certificate pool")
		}
	})
}

func BenchmarkGetCertificateChain(b *testing.B) {
	tempDir := b.TempDir()

	derData, err := pemToDER(validCertPEM)
	if err != nil {
		b.Fatalf("failed to convert PEM to DER: %v", err)
	}

	// Create multiple certificate files
	for i := range 10 {
		certFile := filepath.Join(tempDir, "cert"+string(rune('0'+i))+".crt")
		if err := os.WriteFile(certFile, derData, 0o644); err != nil {
			b.Fatalf("failed to create test certificate: %v", err)
		}
	}

	b.ResetTimer()
	for range b.N {
		_, err := getCertificateChain(tempDir)
		if err != nil {
			b.Fatalf("getCertificateChain failed: %v", err)
		}
	}
}
