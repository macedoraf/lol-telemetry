// Command mocksrv implements a tiny HTTPS server that mimics the League of
// Legends Live Client Data API on port 2999. It is used exclusively by the
// Docker Compose testing environment.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	payloadPath := os.Getenv("MOCK_PAYLOAD")
	if payloadPath == "" {
		payloadPath = "/app/mocks/allgamedata.json"
	}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/liveclientdata/allgamedata", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(payloadPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})

	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	cert, key, err := generateSelfSignedCert()
	if err != nil {
		log.Fatalf("generate cert: %v", err)
	}

	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		log.Fatalf("load cert: %v", err)
	}

	apiServer := &http.Server{
		Addr:    ":2999",
		Handler: apiMux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		},
	}

	healthServer := &http.Server{
		Addr:    ":8080",
		Handler: healthMux,
	}

	go func() {
		log.Println("Health check listening on http://0.0.0.0:8080")
		if err := healthServer.ListenAndServe(); err != nil {
			log.Fatalf("health server: %v", err)
		}
	}()

	log.Println("Mock LoL API listening on https://0.0.0.0:2999")
	if err := apiServer.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("api server: %v", err)
	}
}

func generateSelfSignedCert() (certPEM, keyPEM []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"lol-telemetry-test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:              []string{"mock-api", "localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM, nil
}
