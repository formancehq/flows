package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	sharedapi "github.com/formancehq/go-libs/v2/bun/bunpaginate"

	"github.com/formancehq/go-libs/v2/auth"

	"github.com/formancehq/go-libs/v2/health"
	"github.com/formancehq/orchestration/internal/api"
	v1 "github.com/formancehq/orchestration/internal/api/v1"
	v2 "github.com/formancehq/orchestration/internal/api/v2"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/mock/gomock"
)

func TestModule(t *testing.T) {

	ctrl := gomock.NewController(t)
	backend := api.NewMockBackend(ctrl)

	var mux *chi.Mux
	app := fxtest.New(t,
		auth.Module(auth.ModuleConfig{Enabled: false}),
		fx.Supply(&health.HealthController{}),
		fx.Supply(api.ServiceInfo{}),
		fx.Replace(fx.Annotate(backend, fx.As(new(api.Backend)))),
		fx.NopLogger,
		api.NewModule(testing.Verbose()),
		v1.NewModule(),
		v2.NewModule(),
		fx.Populate(&mux),
	)
	app.RequireStart()

	backend.EXPECT().
		ListWorkflows(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(&sharedapi.Cursor[workflow.Workflow]{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/workflows", nil)
	rsp := httptest.NewRecorder()
	mux.ServeHTTP(rsp, req)
	require.Equal(t, http.StatusOK, rsp.Code)

	req = httptest.NewRequest(http.MethodGet, "/v2/workflows", nil)
	rsp = httptest.NewRecorder()
	mux.ServeHTTP(rsp, req)
	require.Equal(t, http.StatusOK, rsp.Code)
}
