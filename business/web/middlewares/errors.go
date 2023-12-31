package middlewares

import (
	"context"
	"errors"
	"net/http"

	"github.com/so-heil/wishlist/business/validate"
	"github.com/so-heil/wishlist/foundation/web"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func Errors(l *zap.SugaredLogger) web.Middleware {
	m := func(handler web.Handler) web.Handler {
		h := func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			if err := handler(ctx, w, r); err != nil {
				l.Errorln(err)

				var span trace.Span
				ctx, span = web.AddSpan(ctx, "web.request.middlewares.error")
				span.RecordError(err)
				span.End()

				var eue web.EndUserError

				switch {
				case web.IsEndUserError(err):
					errors.As(err, &eue)
				case validate.IsFieldErrors(err):
					var ferr validate.FieldErrors
					errors.As(err, &ferr)
					eue = web.EndUserError{
						Message: "some fileds have invalid values",
						Status:  http.StatusBadRequest,
						Fields:  ferr.Fields(),
					}
				case web.IsExternalError(err):
					eue = web.EndUserError{
						Message: "some services are not available",
						Status:  http.StatusServiceUnavailable,
					}
				default:
					eue = web.EndUserError{
						Message: "something went really wrong",
						Status:  http.StatusInternalServerError,
					}
				}

				if err := web.Respond(w, ctx, eue, eue.Status); err != nil {
					return err
				}
			}

			return nil
		}

		return h
	}

	return m
}
