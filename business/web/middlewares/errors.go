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
		h := func(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
			if err := handler(w, r, ctx); err != nil {
				l.Errorln(err)

				var span trace.Span
				ctx, span = web.AddSpan(ctx, "business.web.request.middlewares.error")
				span.RecordError(err)
				span.End()

				var eue web.EndUserError

				switch err.(type) {
				case web.EndUserError:
					errors.As(err, &eue)
				case validate.FieldErrors:
					var ferr validate.FieldErrors
					errors.As(err, &ferr)
					eue = web.EndUserError{
						Message: "some fileds have invalid values",
						Status:  http.StatusBadRequest,
						Fields:  ferr.Fields(),
					}
				case web.ExternalError:
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

				// If we receive the shutdown err we need to return it
				// back to the base handler to shut down the service.
				// if web.IsShutdown(err) {
				//	return err
				// }
			}

			return nil
		}

		return h
	}

	return m
}
