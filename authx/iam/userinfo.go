package iam

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

type userInfoClient struct {
	endpoint string
	client   *http.Client
}

func newUserInfoClient(cfg Config) *userInfoClient {
	issuer := strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	if issuer == "" {
		return nil
	}

	return &userInfoClient{
		endpoint: issuer + "/protocol/openid-connect/userinfo",
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *userInfoClient) fetch(ctx context.Context, accessToken string) (claimsOut *Claims, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, http.NoBody)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "iam: failed to fetch userinfo")
	}
	defer func() { err = errors.CombineErrors(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Newf("iam: userinfo endpoint returned status %d", resp.StatusCode)
	}

	var out Claims
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, errors.Wrap(err, "iam: failed to decode userinfo response")
	}

	return &out, nil
}
