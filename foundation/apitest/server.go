package apitest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"time"

	"github.com/so-heil/wishlist/business/auth"
	"github.com/so-heil/wishlist/business/keystore"
	"github.com/so-heil/wishlist/business/validate"
	"github.com/so-heil/wishlist/business/web/middlewares"
	"github.com/so-heil/wishlist/foundation/web"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

type APIServerConfig struct {
	KeystoreRotationDur   time.Duration
	KeystoreExpirationDur time.Duration
}

var DefaultAPIServerConfig = APIServerConfig{
	KeystoreRotationDur:   2 * time.Second,
	KeystoreExpirationDur: 3 * time.Second,
}

type APIServer struct {
	Log      *zap.SugaredLogger
	shutdown chan os.Signal
	srv      *httptest.Server
	URL      string
	Auth     *auth.Auth
	App      *web.App
}

func NewAPIServer(config APIServerConfig, l *zap.SugaredLogger) (*APIServer, error) {
	// init validate
	if err := validate.Init(); err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	// init web app
	shutdown := make(chan os.Signal, 1)
	app := web.NewApp(
		l,
		http.NewServeMux(),
		[]web.Middleware{middlewares.Log(l), middlewares.Errors(l)},
		shutdown,
		noop.TracerProvider{}.Tracer("noop"),
	)

	ks, err := keystore.New(config.KeystoreRotationDur, config.KeystoreExpirationDur, shutdown, l)
	if err != nil {
		return nil, fmt.Errorf("create keystore: %w", err)
	}

	srv := httptest.NewServer(app)
	return &APIServer{
		Log:      l,
		shutdown: shutdown,
		srv:      srv,
		URL:      srv.URL,
		Auth:     auth.New(ks),
		App:      app,
	}, nil
}

func (s *APIServer) Close() error {
	s.srv.Close()
	s.shutdown <- syscall.SIGTERM
	// wait for cleanups
	time.Sleep(time.Second)
	return nil
}

func Logger(ignoreLogs bool) (*zap.SugaredLogger, error) {
	var l *zap.SugaredLogger
	if ignoreLogs {
		l = zap.NewNop().Sugar()
	} else {
		log, err := zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("create logger: %w", err)
		}
		l = log.Sugar()
	}

	return l, nil
}
