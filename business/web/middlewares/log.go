package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/so-heil/wishlist/foundation/web"
	"go.uber.org/zap"
)

func Log(log *zap.SugaredLogger) web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			log.Infow("request started", "method", r.Method, "path", r.URL.Path, "traceID", web.GetTraceID(ctx))

			if err := handler(ctx, w, r); err != nil {
				log.Errorw(fmt.Sprintf("request: %s", err), "method", r.Method, "path", r.URL.Path, "traceID", web.GetTraceID(ctx))
				return err
			}

			v := web.GetValues(ctx)
			log.Infow(
				"request ended",
				"method",
				r.Method,
				"path",
				r.URL.Path,
				"traceID",
				web.GetTraceID(ctx),
				"statusCode",
				v.StatusCode,
				"took",
				time.Since(v.Now).String(),
			)
			return nil
		}
	}
}
