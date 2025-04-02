package temporalclient

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/orchestration/internal/tracer"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/fx"
)

func NewModule(address, namespace string, certStr string, key string, initSearchAttributes bool, searchAttributes ...map[string]enums.IndexedValueType) fx.Option {
	return fx.Options(
		fx.Provide(func(logger logging.Logger) (client.Options, error) {

			var cert *tls.Certificate
			if key != "" && certStr != "" {
				clientCert, err := tls.X509KeyPair([]byte(certStr), []byte(key))
				if err != nil {
					return client.Options{}, err
				}
				cert = &clientCert
			}

			tracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
				Tracer: tracer.Tracer,
			})
			if err != nil {
				return client.Options{}, err
			}

			options := client.Options{
				Namespace:    namespace,
				HostPort:     address,
				Interceptors: []interceptor.ClientInterceptor{tracingInterceptor},
				Logger:       newLogger(logger),
			}
			if cert != nil {
				options.ConnectionOptions = client.ConnectionOptions{
					TLS: &tls.Config{Certificates: []tls.Certificate{*cert}},
				}
			}
			return options, nil
		}),
		fx.Provide(client.Dial),
		fx.Invoke(func(lifecycle fx.Lifecycle, c client.Client) {
			lifecycle.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					if initSearchAttributes {
						return CreateSearchAttributes(ctx, c, namespace, searchAttributes...)
					}
					return nil
				},
				OnStop: func(ctx context.Context) error {
					c.Close()
					return nil
				},
			})
		}),
	)
}

func CreateSearchAttributes(ctx context.Context, c client.Client, namespace string, searchAttributes ...map[string]enums.IndexedValueType) error {
	attributes := mergeSearchAttributes(searchAttributes...)

	_, err := c.OperatorService().AddSearchAttributes(logging.TestingContext(), &operatorservice.AddSearchAttributesRequest{
		SearchAttributes: attributes,
		Namespace:        namespace,
	})
	if err != nil {
		if _, ok := err.(*serviceerror.AlreadyExists); !ok {
			return err
		}
	}
	// Search attributes are created asynchronously, so poll the list, until it is ready
	for {
		ret, err := c.OperatorService().ListSearchAttributes(ctx, &operatorservice.ListSearchAttributesRequest{
			Namespace: namespace,
		})
		if err != nil {
			panic(err)
		}

		created := true
		for key := range attributes {
			if ret.CustomAttributes[key] == enums.INDEXED_VALUE_TYPE_UNSPECIFIED {
				created = false
				break
			}
		}

		if created {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func mergeSearchAttributes(searchAttributes ...map[string]enums.IndexedValueType) map[string]enums.IndexedValueType {
	result := make(map[string]enums.IndexedValueType)
	for _, sa := range searchAttributes {
		for k, v := range sa {
			result[k] = v
		}
	}
	return result
}
