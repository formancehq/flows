# openapi

<div align="left">
    <a href="https://www.speakeasy.com/?utm_source=<no value>&utm_campaign=go"><img src="https://custom-icon-badges.demolab.com/badge/-Built%20By%20Speakeasy-212015?style=for-the-badge&logoColor=FBE331&logo=speakeasy&labelColor=545454" /></a>
    <a href="https://opensource.org/licenses/MIT">
        <img src="https://img.shields.io/badge/License-MIT-blue.svg" style="width: 100px; height: 28px;" />
    </a>
</div>


## 🏗 **Welcome to your new SDK!** 🏗

It has been generated successfully based on your OpenAPI spec. However, it is not yet ready for production use. Here are some next steps:
- [ ] 🛠 Make your SDK feel handcrafted by [customizing it](https://www.speakeasy.com/docs/customize-sdks)
- [ ] ♻️ Refine your SDK quickly by iterating locally with the [Speakeasy CLI](https://github.com/speakeasy-api/speakeasy)
- [ ] 🎁 Publish your SDK to package managers by [configuring automatic publishing](https://www.speakeasy.com/docs/advanced-setup/publish-sdks)
- [ ] ✨ When ready to productionize, delete this section from the README

<!-- Start SDK Installation [installation] -->
## SDK Installation

```bash
go get openapi
```
<!-- End SDK Installation [installation] -->

<!-- Start SDK Example Usage [usage] -->
## SDK Example Usage

### Example

```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
)

func main() {
	s := openapi.New(
		openapi.WithSecurity(components.Security{
			ClientID:     "",
			ClientSecret: "",
		}),
	)

	ctx := context.Background()
	res, err := s.Orchestration.V1.GetServerInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```
<!-- End SDK Example Usage [usage] -->

<!-- Start Available Resources and Operations [operations] -->
## Available Resources and Operations

### [Orchestration.V1](docs/sdks/v1/README.md)

* [GetServerInfo](docs/sdks/v1/README.md#getserverinfo) - Get server info
* [ListTriggers](docs/sdks/v1/README.md#listtriggers) - List triggers
* [CreateTrigger](docs/sdks/v1/README.md#createtrigger) - Create trigger
* [ReadTrigger](docs/sdks/v1/README.md#readtrigger) - Read trigger
* [DeleteTrigger](docs/sdks/v1/README.md#deletetrigger) - Delete trigger
* [ListTriggersOccurrences](docs/sdks/v1/README.md#listtriggersoccurrences) - List triggers occurrences
* [ListWorkflows](docs/sdks/v1/README.md#listworkflows) - List registered workflows
* [CreateWorkflow](docs/sdks/v1/README.md#createworkflow) - Create workflow
* [GetWorkflow](docs/sdks/v1/README.md#getworkflow) - Get a flow by id
* [DeleteWorkflow](docs/sdks/v1/README.md#deleteworkflow) - Delete a flow by id
* [RunWorkflow](docs/sdks/v1/README.md#runworkflow) - Run workflow
* [ListInstances](docs/sdks/v1/README.md#listinstances) - List instances of a workflow
* [GetInstance](docs/sdks/v1/README.md#getinstance) - Get a workflow instance by id
* [SendEvent](docs/sdks/v1/README.md#sendevent) - Send an event to a running workflow
* [CancelEvent](docs/sdks/v1/README.md#cancelevent) - Cancel a running workflow
* [GetInstanceHistory](docs/sdks/v1/README.md#getinstancehistory) - Get a workflow instance history by id
* [GetInstanceStageHistory](docs/sdks/v1/README.md#getinstancestagehistory) - Get a workflow instance stage history

### [Orchestration.V2](docs/sdks/v2/README.md)

* [GetServerInfo](docs/sdks/v2/README.md#getserverinfo) - Get server info
* [ListTriggers](docs/sdks/v2/README.md#listtriggers) - List triggers
* [CreateTrigger](docs/sdks/v2/README.md#createtrigger) - Create trigger
* [ReadTrigger](docs/sdks/v2/README.md#readtrigger) - Read trigger
* [DeleteTrigger](docs/sdks/v2/README.md#deletetrigger) - Delete trigger
* [TestTrigger](docs/sdks/v2/README.md#testtrigger) - Test trigger
* [ListTriggersOccurrences](docs/sdks/v2/README.md#listtriggersoccurrences) - List triggers occurrences
* [ListWorkflows](docs/sdks/v2/README.md#listworkflows) - List registered workflows
* [CreateWorkflow](docs/sdks/v2/README.md#createworkflow) - Create workflow
* [GetWorkflow](docs/sdks/v2/README.md#getworkflow) - Get a flow by id
* [DeleteWorkflow](docs/sdks/v2/README.md#deleteworkflow) - Delete a flow by id
* [RunWorkflow](docs/sdks/v2/README.md#runworkflow) - Run workflow
* [ListInstances](docs/sdks/v2/README.md#listinstances) - List instances of a workflow
* [GetInstance](docs/sdks/v2/README.md#getinstance) - Get a workflow instance by id
* [SendEvent](docs/sdks/v2/README.md#sendevent) - Send an event to a running workflow
* [CancelEvent](docs/sdks/v2/README.md#cancelevent) - Cancel a running workflow
* [GetInstanceHistory](docs/sdks/v2/README.md#getinstancehistory) - Get a workflow instance history by id
* [GetInstanceStageHistory](docs/sdks/v2/README.md#getinstancestagehistory) - Get a workflow instance stage history
<!-- End Available Resources and Operations [operations] -->

<!-- Start Retries [retries] -->
## Retries

Some of the endpoints in this SDK support retries. If you use the SDK without any configuration, it will fall back to the default retry strategy provided by the API. However, the default retry strategy can be overridden on a per-operation basis, or across the entire SDK.

To change the default retry strategy for a single API call, simply provide a `retry.Config` object to the call by using the `WithRetries` option:
```go
package main

import (
	"context"
	"log"
	"models/operations"
	"openapi"
	"openapi/models/components"
	"openapi/retry"
)

func main() {
	s := openapi.New(
		openapi.WithSecurity(components.Security{
			ClientID:     "",
			ClientSecret: "",
		}),
	)

	ctx := context.Background()
	res, err := s.Orchestration.V1.GetServerInfo(ctx, operations.WithRetries(
		retry.Config{
			Strategy: "backoff",
			Backoff: &retry.BackoffStrategy{
				InitialInterval: 1,
				MaxInterval:     50,
				Exponent:        1.1,
				MaxElapsedTime:  100,
			},
			RetryConnectionErrors: false,
		}))
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```

If you'd like to override the default retry strategy for all operations that support retries, you can use the `WithRetryConfig` option at SDK initialization:
```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
	"openapi/retry"
)

func main() {
	s := openapi.New(
		openapi.WithRetryConfig(
			retry.Config{
				Strategy: "backoff",
				Backoff: &retry.BackoffStrategy{
					InitialInterval: 1,
					MaxInterval:     50,
					Exponent:        1.1,
					MaxElapsedTime:  100,
				},
				RetryConnectionErrors: false,
			}),
		openapi.WithSecurity(components.Security{
			ClientID:     "",
			ClientSecret: "",
		}),
	)

	ctx := context.Background()
	res, err := s.Orchestration.V1.GetServerInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```
<!-- End Retries [retries] -->

<!-- Start Error Handling [errors] -->
## Error Handling

Handling errors in this SDK should largely match your expectations.  All operations return a response object or an error, they will never return both.  When specified by the OpenAPI spec document, the SDK will return the appropriate subclass.

| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

### Example

```go
package main

import (
	"context"
	"errors"
	"log"
	"openapi"
	"openapi/models/components"
	"openapi/models/sdkerrors"
)

func main() {
	s := openapi.New(
		openapi.WithSecurity(components.Security{
			ClientID:     "",
			ClientSecret: "",
		}),
	)

	ctx := context.Background()
	res, err := s.Orchestration.V1.GetServerInfo(ctx)
	if err != nil {

		var e *sdkerrors.Error
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}

		var e *sdkerrors.SDKError
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}
	}
}

```
<!-- End Error Handling [errors] -->

<!-- Start Server Selection [server] -->
## Server Selection

### Select Server by Index

You can override the default server globally using the `WithServerIndex` option when initializing the SDK client instance. The selected server will then be used as the default on the operations that use it. This table lists the indexes associated with the available servers:

| # | Server | Variables |
| - | ------ | --------- |
| 0 | `http://localhost:8080/` | None |

#### Example

```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
)

func main() {
	s := openapi.New(
		openapi.WithServerIndex(0),
		openapi.WithSecurity(components.Security{
			ClientID:     "",
			ClientSecret: "",
		}),
	)

	ctx := context.Background()
	res, err := s.Orchestration.V1.GetServerInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```


### Override Server URL Per-Client

The default server can also be overridden globally using the `WithServerURL` option when initializing the SDK client instance. For example:
```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
)

func main() {
	s := openapi.New(
		openapi.WithServerURL("http://localhost:8080/"),
		openapi.WithSecurity(components.Security{
			ClientID:     "",
			ClientSecret: "",
		}),
	)

	ctx := context.Background()
	res, err := s.Orchestration.V1.GetServerInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```
<!-- End Server Selection [server] -->

<!-- Start Custom HTTP Client [http-client] -->
## Custom HTTP Client

The Go SDK makes API calls that wrap an internal HTTP client. The requirements for the HTTP client are very simple. It must match this interface:

```go
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
```

The built-in `net/http` client satisfies this interface and a default client based on the built-in is provided by default. To replace this default with a client of your own, you can implement this interface yourself or provide your own client configured as desired. Here's a simple example, which adds a client with a 30 second timeout.

```go
import (
	"net/http"
	"time"
	"github.com/myorg/your-go-sdk"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	sdkClient  = sdk.New(sdk.WithClient(httpClient))
)
```

This can be a convenient way to configure timeouts, cookies, proxies, custom headers, and other low-level configuration.
<!-- End Custom HTTP Client [http-client] -->

<!-- Start Authentication [security] -->
## Authentication

### Per-Client Security Schemes

This SDK supports the following security schemes globally:

| Name           | Type           | Scheme         |
| -------------- | -------------- | -------------- |
| `ClientID`     | oauth2         | OAuth2 token   |
| `ClientSecret` | oauth2         | OAuth2 token   |

You can set the security parameters through the `WithSecurity` option when initializing the SDK client instance. The selected scheme will be used by default to authenticate with the API for all operations that support it. For example:
```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
)

func main() {
	s := openapi.New(
		openapi.WithSecurity(components.Security{
			ClientID:     "",
			ClientSecret: "",
		}),
	)

	ctx := context.Background()
	res, err := s.Orchestration.V1.GetServerInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```
<!-- End Authentication [security] -->

<!-- Start Special Types [types] -->
## Special Types


<!-- End Special Types [types] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This SDK is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

While we value open-source contributions to this SDK, this library is generated programmatically. Any manual changes added to internal files will be overwritten on the next generation. 
We look forward to hearing your feedback. Feel free to open a PR or an issue with a proof of concept and we'll do our best to include it in a future release. 

### SDK Created by [Speakeasy](https://www.speakeasy.com/?utm_source=<no value>&utm_campaign=go)
