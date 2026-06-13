package iam

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
)

// jwtSigner holds an RSA key and signs test JWTs.
type jwtSigner struct {
	privateKey *rsa.PrivateKey
	kid        string
}

func newJWTSigner(t *testing.T) *jwtSigner {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return &jwtSigner{privateKey: key, kid: "test-kid"}
}

func (s *jwtSigner) sign(c *Claims) string {
	headerJSON, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": s.kid})
	payloadJSON, _ := json.Marshal(c)

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)

	signingInput := header + "." + payload
	digest := sha256.Sum256([]byte(signingInput))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, digest[:])

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func (s *jwtSigner) jwksHandler() http.HandlerFunc {
	pub := &s.privateKey.PublicKey
	nBytes := pub.N.Bytes()
	eVal := big.NewInt(int64(pub.E))
	eBytes := eVal.Bytes()

	body, _ := json.Marshal(map[string]interface{}{
		"keys": []map[string]string{
			{
				"kid": s.kid,
				"kty": "RSA",
				"alg": "RS256",
				"use": "sig",
				"n":   base64.RawURLEncoding.EncodeToString(nBytes),
				"e":   base64.RawURLEncoding.EncodeToString(eBytes),
			},
		},
	})

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

func httpClientForHandler(handler http.Handler) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder.Result(), nil
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
