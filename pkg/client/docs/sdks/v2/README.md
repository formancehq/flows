# V2
(*Orchestration.V2*)

### Available Operations

* [GetServerInfo](#getserverinfo) - Get server info
* [ListTriggers](#listtriggers) - List triggers
* [CreateTrigger](#createtrigger) - Create trigger
* [ReadTrigger](#readtrigger) - Read trigger
* [DeleteTrigger](#deletetrigger) - Delete trigger
* [TestTrigger](#testtrigger) - Test trigger
* [ListTriggersOccurrences](#listtriggersoccurrences) - List triggers occurrences
* [ListWorkflows](#listworkflows) - List registered workflows
* [CreateWorkflow](#createworkflow) - Create workflow
* [GetWorkflow](#getworkflow) - Get a flow by id
* [DeleteWorkflow](#deleteworkflow) - Delete a flow by id
* [RunWorkflow](#runworkflow) - Run workflow
* [ListInstances](#listinstances) - List instances of a workflow
* [GetInstance](#getinstance) - Get a workflow instance by id
* [SendEvent](#sendevent) - Send an event to a running workflow
* [CancelEvent](#cancelevent) - Cancel a running workflow
* [GetInstanceHistory](#getinstancehistory) - Get a workflow instance history by id
* [GetInstanceStageHistory](#getinstancestagehistory) - Get a workflow instance stage history

## GetServerInfo

Get server info

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )

    ctx := context.Background()
    res, err := s.Orchestration.V2.GetServerInfo(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2ServerInfo != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |


### Response

**[*operations.V2GetServerInfoResponse](../../models/operations/v2getserverinforesponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## ListTriggers

List triggers

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2ListTriggersRequest{
        Cursor: openapi.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="),
        PageSize: openapi.Int64(100),
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.ListTriggers(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2ListTriggersResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                            | Type                                                                                 | Required                                                                             | Description                                                                          |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| `ctx`                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                | :heavy_check_mark:                                                                   | The context to use for the request.                                                  |
| `request`                                                                            | [operations.V2ListTriggersRequest](../../models/operations/v2listtriggersrequest.md) | :heavy_check_mark:                                                                   | The request object to use for the request.                                           |
| `opts`                                                                               | [][operations.Option](../../models/operations/option.md)                             | :heavy_minus_sign:                                                                   | The options for this request.                                                        |


### Response

**[*operations.V2ListTriggersResponse](../../models/operations/v2listtriggersresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## CreateTrigger

Create trigger

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )

    ctx := context.Background()
    res, err := s.Orchestration.V2.CreateTrigger(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2CreateTriggerResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                            | Type                                                                 | Required                                                             | Description                                                          |
| -------------------------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------- |
| `ctx`                                                                | [context.Context](https://pkg.go.dev/context#Context)                | :heavy_check_mark:                                                   | The context to use for the request.                                  |
| `request`                                                            | [components.V2TriggerData](../../models/components/v2triggerdata.md) | :heavy_check_mark:                                                   | The request object to use for the request.                           |
| `opts`                                                               | [][operations.Option](../../models/operations/option.md)             | :heavy_minus_sign:                                                   | The options for this request.                                        |


### Response

**[*operations.V2CreateTriggerResponse](../../models/operations/v2createtriggerresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## ReadTrigger

Read trigger

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2ReadTriggerRequest{
        TriggerID: "<value>",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.ReadTrigger(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2ReadTriggerResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.V2ReadTriggerRequest](../../models/operations/v2readtriggerrequest.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |
| `opts`                                                                             | [][operations.Option](../../models/operations/option.md)                           | :heavy_minus_sign:                                                                 | The options for this request.                                                      |


### Response

**[*operations.V2ReadTriggerResponse](../../models/operations/v2readtriggerresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## DeleteTrigger

Read trigger

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2DeleteTriggerRequest{
        TriggerID: "<value>",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.DeleteTrigger(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                              | Type                                                                                   | Required                                                                               | Description                                                                            |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `ctx`                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                  | :heavy_check_mark:                                                                     | The context to use for the request.                                                    |
| `request`                                                                              | [operations.V2DeleteTriggerRequest](../../models/operations/v2deletetriggerrequest.md) | :heavy_check_mark:                                                                     | The request object to use for the request.                                             |
| `opts`                                                                                 | [][operations.Option](../../models/operations/option.md)                               | :heavy_minus_sign:                                                                     | The options for this request.                                                          |


### Response

**[*operations.V2DeleteTriggerResponse](../../models/operations/v2deletetriggerresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## TestTrigger

Test trigger

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.TestTriggerRequest{
        TriggerID: "<value>",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.TestTrigger(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2TestTriggerResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.TestTriggerRequest](../../models/operations/testtriggerrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |


### Response

**[*operations.TestTriggerResponse](../../models/operations/testtriggerresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## ListTriggersOccurrences

List triggers occurrences

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2ListTriggersOccurrencesRequest{
        TriggerID: "<value>",
        Cursor: openapi.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="),
        PageSize: openapi.Int64(100),
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.ListTriggersOccurrences(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2ListTriggersOccurrencesResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                  | Type                                                                                                       | Required                                                                                                   | Description                                                                                                |
| ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                                      | :heavy_check_mark:                                                                                         | The context to use for the request.                                                                        |
| `request`                                                                                                  | [operations.V2ListTriggersOccurrencesRequest](../../models/operations/v2listtriggersoccurrencesrequest.md) | :heavy_check_mark:                                                                                         | The request object to use for the request.                                                                 |
| `opts`                                                                                                     | [][operations.Option](../../models/operations/option.md)                                                   | :heavy_minus_sign:                                                                                         | The options for this request.                                                                              |


### Response

**[*operations.V2ListTriggersOccurrencesResponse](../../models/operations/v2listtriggersoccurrencesresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## ListWorkflows

List registered workflows

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2ListWorkflowsRequest{
        Cursor: openapi.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="),
        PageSize: openapi.Int64(100),
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.ListWorkflows(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2ListWorkflowsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                              | Type                                                                                   | Required                                                                               | Description                                                                            |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `ctx`                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                  | :heavy_check_mark:                                                                     | The context to use for the request.                                                    |
| `request`                                                                              | [operations.V2ListWorkflowsRequest](../../models/operations/v2listworkflowsrequest.md) | :heavy_check_mark:                                                                     | The request object to use for the request.                                             |
| `opts`                                                                                 | [][operations.Option](../../models/operations/option.md)                               | :heavy_minus_sign:                                                                     | The options for this request.                                                          |


### Response

**[*operations.V2ListWorkflowsResponse](../../models/operations/v2listworkflowsresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## CreateWorkflow

Create a workflow

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )

    ctx := context.Background()
    res, err := s.Orchestration.V2.CreateWorkflow(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2CreateWorkflowResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `request`                                                                                | [components.V2CreateWorkflowRequest](../../models/components/v2createworkflowrequest.md) | :heavy_check_mark:                                                                       | The request object to use for the request.                                               |
| `opts`                                                                                   | [][operations.Option](../../models/operations/option.md)                                 | :heavy_minus_sign:                                                                       | The options for this request.                                                            |


### Response

**[*operations.V2CreateWorkflowResponse](../../models/operations/v2createworkflowresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetWorkflow

Get a flow by id

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2GetWorkflowRequest{
        FlowID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.GetWorkflow(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2GetWorkflowResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.V2GetWorkflowRequest](../../models/operations/v2getworkflowrequest.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |
| `opts`                                                                             | [][operations.Option](../../models/operations/option.md)                           | :heavy_minus_sign:                                                                 | The options for this request.                                                      |


### Response

**[*operations.V2GetWorkflowResponse](../../models/operations/v2getworkflowresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## DeleteWorkflow

Delete a flow by id

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2DeleteWorkflowRequest{
        FlowID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.DeleteWorkflow(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `request`                                                                                | [operations.V2DeleteWorkflowRequest](../../models/operations/v2deleteworkflowrequest.md) | :heavy_check_mark:                                                                       | The request object to use for the request.                                               |
| `opts`                                                                                   | [][operations.Option](../../models/operations/option.md)                                 | :heavy_minus_sign:                                                                       | The options for this request.                                                            |


### Response

**[*operations.V2DeleteWorkflowResponse](../../models/operations/v2deleteworkflowresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## RunWorkflow

Run workflow

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2RunWorkflowRequest{
        WorkflowID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.RunWorkflow(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2RunWorkflowResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.V2RunWorkflowRequest](../../models/operations/v2runworkflowrequest.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |
| `opts`                                                                             | [][operations.Option](../../models/operations/option.md)                           | :heavy_minus_sign:                                                                 | The options for this request.                                                      |


### Response

**[*operations.V2RunWorkflowResponse](../../models/operations/v2runworkflowresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## ListInstances

List instances of a workflow

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2ListInstancesRequest{
        Cursor: openapi.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="),
        PageSize: openapi.Int64(100),
        WorkflowID: openapi.String("xxx"),
        Running: openapi.Bool(true),
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.ListInstances(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2ListRunsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                              | Type                                                                                   | Required                                                                               | Description                                                                            |
| -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `ctx`                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                  | :heavy_check_mark:                                                                     | The context to use for the request.                                                    |
| `request`                                                                              | [operations.V2ListInstancesRequest](../../models/operations/v2listinstancesrequest.md) | :heavy_check_mark:                                                                     | The request object to use for the request.                                             |
| `opts`                                                                                 | [][operations.Option](../../models/operations/option.md)                               | :heavy_minus_sign:                                                                     | The options for this request.                                                          |


### Response

**[*operations.V2ListInstancesResponse](../../models/operations/v2listinstancesresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetInstance

Get a workflow instance by id

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2GetInstanceRequest{
        InstanceID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.GetInstance(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2GetWorkflowInstanceResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.V2GetInstanceRequest](../../models/operations/v2getinstancerequest.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |
| `opts`                                                                             | [][operations.Option](../../models/operations/option.md)                           | :heavy_minus_sign:                                                                 | The options for this request.                                                      |


### Response

**[*operations.V2GetInstanceResponse](../../models/operations/v2getinstanceresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## SendEvent

Send an event to a running workflow

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2SendEventRequest{
        InstanceID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.SendEvent(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.V2SendEventRequest](../../models/operations/v2sendeventrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |


### Response

**[*operations.V2SendEventResponse](../../models/operations/v2sendeventresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## CancelEvent

Cancel a running workflow

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2CancelEventRequest{
        InstanceID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.CancelEvent(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.V2CancelEventRequest](../../models/operations/v2canceleventrequest.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |
| `opts`                                                                             | [][operations.Option](../../models/operations/option.md)                           | :heavy_minus_sign:                                                                 | The options for this request.                                                      |


### Response

**[*operations.V2CancelEventResponse](../../models/operations/v2canceleventresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetInstanceHistory

Get a workflow instance history by id

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2GetInstanceHistoryRequest{
        InstanceID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.GetInstanceHistory(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2GetWorkflowInstanceHistoryResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                        | Type                                                                                             | Required                                                                                         | Description                                                                                      |
| ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                            | [context.Context](https://pkg.go.dev/context#Context)                                            | :heavy_check_mark:                                                                               | The context to use for the request.                                                              |
| `request`                                                                                        | [operations.V2GetInstanceHistoryRequest](../../models/operations/v2getinstancehistoryrequest.md) | :heavy_check_mark:                                                                               | The request object to use for the request.                                                       |
| `opts`                                                                                           | [][operations.Option](../../models/operations/option.md)                                         | :heavy_minus_sign:                                                                               | The options for this request.                                                                    |


### Response

**[*operations.V2GetInstanceHistoryResponse](../../models/operations/v2getinstancehistoryresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetInstanceStageHistory

Get a workflow instance stage history

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.V2GetInstanceStageHistoryRequest{
        InstanceID: "xxx",
        Number: 0,
    }
    ctx := context.Background()
    res, err := s.Orchestration.V2.GetInstanceStageHistory(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.V2GetWorkflowInstanceHistoryStageResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                  | Type                                                                                                       | Required                                                                                                   | Description                                                                                                |
| ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                      | [context.Context](https://pkg.go.dev/context#Context)                                                      | :heavy_check_mark:                                                                                         | The context to use for the request.                                                                        |
| `request`                                                                                                  | [operations.V2GetInstanceStageHistoryRequest](../../models/operations/v2getinstancestagehistoryrequest.md) | :heavy_check_mark:                                                                                         | The request object to use for the request.                                                                 |
| `opts`                                                                                                     | [][operations.Option](../../models/operations/option.md)                                                   | :heavy_minus_sign:                                                                                         | The options for this request.                                                                              |


### Response

**[*operations.V2GetInstanceStageHistoryResponse](../../models/operations/v2getinstancestagehistoryresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.V2Error  | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |
