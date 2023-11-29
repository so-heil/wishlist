package probes

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/so-heil/wishlist/foundation/web"
	"go.uber.org/zap"
)

type Probes struct {
	log *zap.SugaredLogger
	app *web.App
}

func New(log *zap.SugaredLogger, app *web.App) *Probes {
	return &Probes{log: log, app: app}
}

func (p *Probes) readiness(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	status := "ok"
	statusCode := http.StatusOK

	data := struct {
		Status string `json:"status"`
	}{
		Status: status,
	}

	return web.Respond(w, ctx, data, statusCode)
}

// TODO: this stuff should not be accessible publicly
func (p *Probes) liveness(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
	host, err := os.Hostname()
	if err != nil {
		host = "unavailable"
	}

	data := struct {
		Status     string `json:"status,omitempty"`
		Host       string `json:"host,omitempty"`
		Name       string `json:"name,omitempty"`
		PodIP      string `json:"podIP,omitempty"`
		Node       string `json:"node,omitempty"`
		Namespace  string `json:"namespace,omitempty"`
		GOMAXPROCS int    `json:"GOMAXPROCS,omitempty"`
	}{
		Status:     "up",
		Host:       host,
		Name:       os.Getenv("KUBERNETES_NAME"),
		PodIP:      os.Getenv("KUBERNETES_POD_IP"),
		Node:       os.Getenv("KUBERNETES_NODE_NAME"),
		Namespace:  os.Getenv("KUBERNETES_NAMESPACE"),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
	}

	return web.Respond(w, ctx, data, http.StatusOK)
}

func (p *Probes) Routes(group string) {
	p.app.Handle(http.MethodGet, group, "/liveness", p.liveness)
	p.app.Handle(http.MethodGet, group, "/readiness", p.readiness)
}
