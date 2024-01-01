package middlewares

import (
	"context"
	"net/http"

	"github.com/so-heil/wishlist/business/auth"
	"github.com/so-heil/wishlist/foundation/web"
)

func Auth(a *auth.Auth) web.Middleware {
	m := func(handler web.Handler) web.Handler {
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			authHeader := r.Header.Get("Authorization")
			var uc auth.UserClaims
			if err := a.ParseFromBearer(authHeader, &uc); err != nil {
				return web.EUEFromError(auth.ErrInvalidToken, http.StatusUnauthorized)
			}

			ctx = auth.SetUserID(ctx, uc.ID)
			return handler(ctx, w, r)
		}
		return h
	}
	return m
}
