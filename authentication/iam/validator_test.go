package iam

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// base64EncodeJSON marshals v to JSON and returns the base64url-encoded string.
func base64EncodeJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return base64.RawURLEncoding.EncodeToString(b)
}

// splitToken splits a JWT string into its three dot-separated parts.
func splitToken(token string) [3]string {
	parts := strings.SplitN(token, ".", 3)
	return [3]string{parts[0], parts[1], parts[2]}
}

type ValidatorSuite struct {
	suite.Suite
	serverURL string
	signer    *jwtSigner
	validator *validator
}

func TestValidatorSuite(t *testing.T) {
	suite.Run(t, new(ValidatorSuite))
}

func (s *ValidatorSuite) SetupSuite() {
	s.signer = newJWTSigner(s.T())
}

// newValidator builds a validator with the given in-process HTTP client and no JWKS cooldown (tests construct fresh
// validators per case).
func (s *ValidatorSuite) newValidator(client *http.Client) *validator {
	return newValidator(s.testConfig(), newOptions([]Option{
		WithHTTPClient(client),
		WithJWKSCooldown(0),
	}))
}

func (s *ValidatorSuite) testConfig() Config {
	return Config{
		JwksURL:     s.serverURL + "/certs",
		Issuer:      "https://iam.example.com/realms/tupic",
		ServiceName: "Tests",
	}
}

func (s *ValidatorSuite) SetupTest() {
	s.serverURL = "https://iam.test"

	s.validator = s.newValidator(httpClientForHandler(s.signer.jwksHandler()))
}

func (s *ValidatorSuite) TestValidToken_ReturnsClaims() {
	sub := uuid.New().String()
	c := Claims{
		Sub:   sub,
		Iss:   "https://iam.example.com/realms/tupic",
		Email: "user@example.com",
		Exp:   time.Now().Add(time.Hour).Unix(),
	}
	token := s.signer.sign(&c)

	got, err := s.validator.validate(context.Background(), token)
	s.Require().NoError(err)
	s.Equal(sub, got.Sub)
	s.Equal("user@example.com", got.Email)
}

func (s *ValidatorSuite) TestZeroExp_IsValid() {
	c := Claims{
		Sub: uuid.New().String(),
		Iss: "https://iam.example.com/realms/tupic",
		Exp: 0,
	}
	token := s.signer.sign(&c)

	_, err := s.validator.validate(context.Background(), token)
	s.NoError(err)
}

func (s *ValidatorSuite) TestExpiredToken_ReturnsError() {
	c := Claims{
		Sub: uuid.New().String(),
		Iss: "https://iam.example.com/realms/tupic",
		Exp: time.Now().Add(-time.Hour).Unix(),
	}
	token := s.signer.sign(&c)

	_, err := s.validator.validate(context.Background(), token)
	s.Require().Error(err)
	s.Contains(err.Error(), "expired")
}

func (s *ValidatorSuite) TestUnsupportedAlgorithm_ReturnsError() {
	// Manually construct a JWT with HS256 in the header.
	header := base64EncodeJSON(map[string]string{"alg": "HS256", "kid": s.signer.kid})
	payload := base64EncodeJSON(map[string]string{
		"sub": uuid.New().String(),
		"iss": "https://iam.example.com/realms/tupic",
	})
	token := header + "." + payload + ".invalidsig"

	_, err := s.validator.validate(context.Background(), token)
	s.Require().Error(err)
	s.Contains(err.Error(), "unsupported JWT algorithm")
}

func (s *ValidatorSuite) TestInvalidFormat_ReturnsError() {
	_, err := s.validator.validate(context.Background(), "only.two")
	s.Require().Error(err)
	s.Contains(err.Error(), "invalid JWT format")
}

func (s *ValidatorSuite) TestUnknownKid_ReturnsError() {
	// Use a different signer with a different kid so it won't be in the JWKS.
	otherSigner := newJWTSigner(s.T())
	otherSigner.kid = "unknown-kid"

	c := Claims{Sub: uuid.New().String(), Iss: "https://iam.example.com/realms/tupic"}
	token := otherSigner.sign(&c)

	_, err := s.validator.validate(context.Background(), token)
	s.Require().Error(err)
	s.Contains(err.Error(), "key not found")
}

func (s *ValidatorSuite) TestTamperedPayload_ReturnsError() {
	c := Claims{
		Sub: uuid.New().String(),
		Iss: "https://iam.example.com/realms/tupic",
		Exp: time.Now().Add(time.Hour).Unix(),
	}
	token := s.signer.sign(&c)

	// Replace the payload part with a different base64 value while keeping the original signature.
	parts := splitToken(token)
	tamperedPayload := base64EncodeJSON(map[string]string{
		"sub": "attacker-id",
		"iss": "https://iam.example.com/realms/tupic",
	})
	tampered := parts[0] + "." + tamperedPayload + "." + parts[2]

	_, err := s.validator.validate(context.Background(), tampered)
	s.Require().Error(err)
	s.Contains(err.Error(), "verification failed")
}

func (s *ValidatorSuite) TestJWKSServerError_ReturnsError() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	v := s.newValidator(httpClientForHandler(handler))

	c := Claims{Sub: uuid.New().String(), Iss: "https://iam.example.com/realms/tupic"}
	token := s.signer.sign(&c)

	_, err := v.validate(context.Background(), token)
	s.Require().Error(err)
}

func (s *ValidatorSuite) TestKeyCache_DoesNotRefetch() {
	requestCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		s.signer.jwksHandler()(w, r)
	})

	v := s.newValidator(httpClientForHandler(handler))

	c := Claims{
		Sub: uuid.New().String(),
		Iss: "https://iam.example.com/realms/tupic",
		Exp: time.Now().Add(time.Hour).Unix(),
	}
	token := s.signer.sign(&c)

	_, err := v.validate(context.Background(), token)
	s.Require().NoError(err)

	_, err = v.validate(context.Background(), token)
	s.Require().NoError(err)

	s.Equal(1, requestCount, "JWKS endpoint should only be fetched once due to caching")
}

func (s *ValidatorSuite) TestIssuerMismatch_ReturnsError() {
	c := Claims{
		Sub: uuid.New().String(),
		Iss: "https://iam.example.com/realms/other",
		Exp: time.Now().Add(time.Hour).Unix(),
	}
	token := s.signer.sign(&c)

	_, err := s.validator.validate(context.Background(), token)
	s.Require().Error(err)
	s.Contains(err.Error(), "issuer mismatch")
}

func (s *ValidatorSuite) TestMissingSubject_ReturnsError() {
	c := Claims{
		Iss: "https://iam.example.com/realms/tupic",
		Exp: time.Now().Add(time.Hour).Unix(),
	}
	token := s.signer.sign(&c)

	_, err := s.validator.validate(context.Background(), token)
	s.Require().Error(err)
	s.Contains(err.Error(), "subject is required")
}

// TestUnknownKid_CooldownPreventsRefetch verifies that repeated unknown key IDs do not hammer the JWKS endpoint within
// the cooldown window.
func (s *ValidatorSuite) TestUnknownKid_CooldownPreventsRefetch() {
	requestCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		s.signer.jwksHandler()(w, r)
	})

	v := newValidator(s.testConfig(), newOptions([]Option{
		WithHTTPClient(httpClientForHandler(handler)),
		WithJWKSCooldown(time.Hour),
	}))

	otherSigner := newJWTSigner(s.T())
	otherSigner.kid = "rotating-kid"
	c := Claims{Sub: uuid.New().String(), Iss: "https://iam.example.com/realms/tupic"}
	token := otherSigner.sign(&c)

	for range 5 {
		_, err := v.validate(context.Background(), token)
		s.Require().Error(err)
	}

	s.Equal(1, requestCount, "JWKS endpoint must be fetched at most once within the cooldown")
}
