package api

import (
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
)

type ServiceInfo struct {
	Version string `json:"version"`
}

func getInfo(info ServiceInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api.RawOk(w, info)
	}
}
