// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// oidcHTTPTimeout bounds every request to the identity provider, so a hanging
// provider cannot block OIDC logins indefinitely.
const oidcHTTPTimeout = 10 * time.Second

var errInvalidBoolClaim = errors.New("invalid boolean claim")

// oidcBool accepts a JSON boolean or a "true"/"false" string, since some
// providers (e.g. AWS Cognito) send email_verified as a string.
type oidcBool bool

func (b *oidcBool) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "true", `"true"`:
		*b = true
	case "false", `"false"`:
		*b = false
	default:
		return errInvalidBoolClaim
	}

	return nil
}

// oidcEmailClaims holds the e-mail claims of an ID token or a UserInfo response.
type oidcEmailClaims struct {
	Email         string    `json:"email"`
	EmailVerified *oidcBool `json:"email_verified"`
}

// verified reports whether the provider sent email_verified set to true.
func (c oidcEmailClaims) verified() bool {
	return c.EmailVerified != nil && bool(*c.EmailVerified)
}

// oidcIdentityID scopes the subject to its issuer, since sub is unique only within one issuer.
// An issuer URL contains no fragment, so "#" separates issuer and subject unambiguously.
func oidcIdentityID(issuer, subject string) string {
	return fmt.Sprintf("%s#%s", issuer, subject)
}

// OIDCConfig holds configuration of a generic OpenID Connect provider.
// The OAuth endpoints are resolved from the issuer's discovery document,
// so OAuth.Endpoint is ignored.
type OIDCConfig struct {
	IssuerURL string
	OAuth     oauth2.Config
}

type oidcClient struct {
	cfg        OIDCConfig
	httpClient *http.Client
	mu         sync.Mutex
	provider   *oidc.Provider
}

func newOIDCClient(cfg OIDCConfig) *oidcClient {
	return &oidcClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: oidcHTTPTimeout,
		},
	}
}

func (c *oidcClient) enabled() bool {
	return c.cfg.IssuerURL != ""
}

// httpContext makes go-oidc and oauth2 calls made with ctx use the client with a timeout.
func (c *oidcClient) httpContext(ctx context.Context) context.Context {
	return oidc.ClientContext(ctx, c.httpClient)
}

// discover fetches the issuer's discovery document on first use and caches it.
// A failed discovery is retried on the next call, so the service can start
// while the identity provider is unavailable.
func (c *oidcClient) discover() (*oidc.Provider, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.provider != nil {
		return c.provider, nil
	}

	// go-oidc keeps this context for later JWKS refreshes, so it must not be request scoped.
	provider, err := oidc.NewProvider(c.httpContext(context.Background()), c.cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	c.provider = provider
	return provider, nil
}

func (c *oidcClient) oauthConfig() (oauth2.Config, error) {
	provider, err := c.discover()
	if err != nil {
		return oauth2.Config{}, err
	}

	cfg := c.cfg.OAuth
	cfg.Endpoint = provider.Endpoint()
	return cfg, nil
}

// emailClaims returns the e-mail claims of the ID token,
// or of UserInfo when the ID token has no e-mail.
func (c *oidcClient) emailClaims(ctx context.Context, idToken *oidc.IDToken, oauthToken *oauth2.Token) (oidcEmailClaims, error) {
	var claims oidcEmailClaims
	if err := idToken.Claims(&claims); err != nil {
		return oidcEmailClaims{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	if claims.Email != "" {
		return claims, nil
	}

	// Some providers only expose the e-mail through the UserInfo endpoint.
	provider, err := c.discover()
	if err != nil {
		return oidcEmailClaims{}, err
	}

	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauthToken))
	if err != nil {
		return oidcEmailClaims{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	// UserInfo must describe the same user as the ID token.
	if userInfo.Subject != idToken.Subject {
		return oidcEmailClaims{}, errors.ErrAuthentication
	}

	// UserInfo claims replace the ID token claims, so email_verified describes the UserInfo e-mail.
	var userInfoClaims oidcEmailClaims
	if err := userInfo.Claims(&userInfoClaims); err != nil {
		return oidcEmailClaims{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	return userInfoClaims, nil
}

func (c *oidcClient) verifier() (*oidc.IDTokenVerifier, error) {
	provider, err := c.discover()
	if err != nil {
		return nil, err
	}

	verifierConfig := &oidc.Config{
		ClientID: c.cfg.OAuth.ClientID,
	}
	return provider.Verifier(verifierConfig), nil
}
