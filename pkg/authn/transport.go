package authn

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/go-kit/kit/endpoint"
)

// HTTPTokenToContext is a go-kit RequestFunc that extracts the authentication token from the HTTP header and returns a context derived
// from ctx with the token value attached under the tokenKey{} key.
func HTTPTokenToContext(ctx context.Context, req *http.Request) context.Context {
	if token := apiutil.ExtractBearerToken(req); token != "" {
		return context.WithValue(ctx, tokenCtxKey{}, token)
	}

	return ctx
}

// IdentityMiddleware returns a go-kit endpoint Middleware that attaches a domain.Identity associated with the authentication
// token of the current request to the current context.
func IdentityMiddleware(auth domain.AuthClient, log logger.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request any) (any, error) {
			token, _ := ctx.Value(tokenCtxKey{}).(string)
			if token != "" {
				identity, err := auth.Identify(ctx, token)
				if err != nil {
					log.Warn(fmt.Sprintf("error decoding token to identity: %v", err))
					return next(ctx, request)
				}

				ctx = context.WithValue(ctx, identityCtxKey{}, identity)
			}

			return next(ctx, request)
		}
	}
}
