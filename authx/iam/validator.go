package iam

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
)

// jwtHeader represents the decoded header portion of a JWT token containing key
// ID and algorithm information.
type jwtHeader struct {
	Kid string `json:"kid"`
	Alg string `json:"alg"`
}

// jwk represents a JSON Web Key containing cryptographic key material and
// metadata for signature verification.
type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// jwksResponse represents the JSON Web Key Set response containing a collection
// of public keys from the JWKS endpoint.
type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

// validator validates JWT tokens using RS256 and a JWKS endpoint.
type validator struct {
	cfg    Config
	mu     sync.RWMutex
	keys   map[string]*rsa.PublicKey
	client *http.Client
}

// newValidator creates and initializes a new validator instance with the
// provided configuration.
func newValidator(cfg Config) *validator {
	return &validator{
		cfg:    cfg,
		keys:   make(map[string]*rsa.PublicKey),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// validate verifies a JWT token string using RS256 signature validation and
// returns the parsed claims.
func (v *validator) validate(ctx context.Context, tokenString string) (*Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format: expected 3 parts")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode JWT header")
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, errors.Wrap(err, "failed to parse JWT header")
	}
	if header.Alg != "RS256" {
		return nil, errors.Newf("unsupported JWT algorithm: %s", header.Alg)
	}
	publicKey, err := v.getKey(ctx, header.Kid)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	signingInput := parts[0] + "." + parts[1]
	digest := sha256.Sum256([]byte(signingInput))
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode JWT signature")
	}
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, digest[:], sigBytes); err != nil {
		return nil, errors.Wrap(err, "JWT signature verification failed")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode JWT payload")
	}
	var c Claims
	if err := json.Unmarshal(payloadBytes, &c); err != nil {
		return nil, errors.Wrap(err, "failed to parse JWT claims")
	}
	if strings.TrimSpace(c.Sub) == "" {
		return nil, errors.New("JWT subject is required")
	}
	if expectedIssuer := strings.TrimSpace(v.cfg.Issuer); expectedIssuer != "" && c.Iss != expectedIssuer {
		return nil, errors.Newf("JWT issuer mismatch: got %q", c.Iss)
	}
	if c.Exp > 0 && c.Exp < time.Now().Unix() {
		return nil, errors.New("JWT token has expired")
	}
	return &c, nil
}

// getKey retrieves an RSA public key by kid from the cache, fetching keys from
// JWKS endpoint if not cached.
func (v *validator) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	v.mu.RUnlock()
	if ok {
		return key, nil
	}
	if err := v.fetchKeys(ctx); err != nil {
		return nil, errors.WithStack(err)
	}
	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()
	if !ok {
		return nil, errors.Newf("JWT key not found for kid: %s", kid)
	}
	return key, nil
}

// fetchKeys retrieves and parses RSA public keys from the JWKS endpoint and
// updates the validator's key cache.
func (v *validator) fetchKeys(ctx context.Context) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.JwksURL, http.NoBody)
	if err != nil {
		return errors.WithStack(err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to fetch JWKS")
	}
	defer func() { err = errors.CombineErrors(err, resp.Body.Close()) }()
	if resp.StatusCode != http.StatusOK {
		return errors.Newf("JWKS endpoint returned status %d", resp.StatusCode)
	}
	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return errors.Wrap(err, "failed to decode JWKS response")
	}
	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return errors.Wrapf(err, "failed to decode RSA modulus for kid %s", k.Kid)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return errors.Wrapf(err, "failed to decode RSA exponent for kid %s", k.Kid)
		}
		keys[k.Kid] = &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(new(big.Int).SetBytes(eBytes).Int64()),
		}
	}
	v.mu.Lock()
	v.keys = keys
	v.mu.Unlock()
	return nil
}
