package api

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/formancehq/go-libs/v3/auth"
	"github.com/formancehq/go-libs/v3/health"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

func TagVersion() fx.Annotation {
	return fx.ResultTags(`group:"apiVersions"`)
}

func NewModule(debug bool) fx.Option {
	return fx.Options(
		fx.Provide(fx.Annotate(func(
			backend Backend,
			info ServiceInfo,
			healthController *health.HealthController,
			authenticator auth.Authenticator,
			publisher message.Publisher,
			versions ...Version) *chi.Mux {
			return NewRouter(backend, info, healthController, authenticator, publisher, debug, versions...)
		}, fx.ParamTags(``, ``, ``, ``, ``, `group:"apiVersions"`))),
		fx.Provide(NewDefaultBackend),
	)
}
