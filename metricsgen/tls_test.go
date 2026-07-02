// Copyright 2022 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metricsgen

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"
)

// generateSelfSignedCert writes a self-signed PEM certificate and key to the
// provided file paths.
func generateSelfSignedCert(t *testing.T, certFile, keyFile string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	cf, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	defer cf.Close()
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatalf("encode cert: %v", err)
	}

	kf, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	defer kf.Close()
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	if err := pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		t.Fatalf("encode key: %v", err)
	}
}

func TestBuildTLSConfig(t *testing.T) {
	dir := t.TempDir()
	certFile := dir + "/client.crt"
	keyFile := dir + "/client.key"
	caFile := dir + "/ca.crt"
	caKeyFile := dir + "/ca.key"

	generateSelfSignedCert(t, certFile, keyFile)
	generateSelfSignedCert(t, caFile, caKeyFile)

	tests := []struct {
		name     string
		insecure bool
		certFile string
		keyFile  string
		caFile   string
		wantErr  bool
	}{
		{
			name:     "defaults: no TLS material",
			insecure: false,
			wantErr:  false,
		},
		{
			name:     "insecure skip verify",
			insecure: true,
			wantErr:  false,
		},
		{
			name:     "client cert and key",
			certFile: certFile,
			keyFile:  keyFile,
			wantErr:  false,
		},
		{
			name:    "CA cert only",
			caFile:  caFile,
			wantErr: false,
		},
		{
			name:     "all three flags",
			insecure: false,
			certFile: certFile,
			keyFile:  keyFile,
			caFile:   caFile,
			wantErr:  false,
		},
		{
			name:     "insecure plus CA file (CA ignored, no error)",
			insecure: true,
			caFile:   caFile,
			wantErr:  false,
		},
		{
			name:     "cert without key",
			certFile: certFile,
			wantErr:  true,
		},
		{
			name:    "key without cert",
			keyFile: keyFile,
			wantErr: true,
		},
		{
			name:     "nonexistent cert file",
			certFile: dir + "/no.crt",
			keyFile:  keyFile,
			wantErr:  true,
		},
		{
			name:     "nonexistent key file",
			certFile: certFile,
			keyFile:  dir + "/no.key",
			wantErr:  true,
		},
		{
			name:    "nonexistent CA file",
			caFile:  dir + "/no-ca.crt",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := buildTLSConfig(tc.insecure, tc.certFile, tc.keyFile, tc.caFile)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("expected non-nil tls.Config, got nil")
			}

			if cfg.InsecureSkipVerify != tc.insecure {
				t.Errorf("InsecureSkipVerify: got %v, want %v", cfg.InsecureSkipVerify, tc.insecure)
			}

			wantCerts := tc.certFile != ""
			if wantCerts && len(cfg.Certificates) != 1 {
				t.Errorf("expected exactly 1 client certificate, got %d", len(cfg.Certificates))
			}
			if !wantCerts && len(cfg.Certificates) != 0 {
				t.Errorf("expected no client certificates, got %d", len(cfg.Certificates))
			}

			wantCA := tc.caFile != ""
			if wantCA && cfg.RootCAs == nil {
				t.Error("expected RootCAs to be set, got nil")
			}
			if !wantCA && cfg.RootCAs != nil {
				t.Error("expected RootCAs to be nil")
			}
		})
	}
}
