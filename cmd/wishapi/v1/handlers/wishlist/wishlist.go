package wishlist

import (
	"context"
	"net/http"

	"github.com/so-heil/wishlist/foundation/web"
)

type Wishlist struct {
	app *web.App
}

func New(app *web.App) *Wishlist {
	return &Wishlist{app: app}
}

func Get(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
	return web.Respond(w, ctx, struct {
		Ok bool `json:"ok"`
	}{Ok: true}, http.StatusOK)
}

func (wl *Wishlist) HandleRoutes(group string) {
	wl.app.Handle(http.MethodGet, group, "/get", Get, nil)
}
