# V1
(*Orchestration.V1*)

### Available Operations

* [GetServerInfo](#getserverinfo) - Get server info
* [ListTriggers](#listtriggers) - List triggers
* [CreateTrigger](#createtrigger) - Create trigger
* [ReadTrigger](#readtrigger) - Read trigger
* [DeleteTrigger](#deletetrigger) - Delete trigger
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
    res, err := s.Orchestration.V1.GetServerInfo(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.ServerInfo != nil {
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

**[*operations.GetServerInfoResponse](../../models/operations/getserverinforesponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.ListTriggersRequest{}
    ctx := context.Background()
    res, err := s.Orchestration.V1.ListTriggers(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListTriggersResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                        | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ctx`                                                                            | [context.Context](https://pkg.go.dev/context#Context)                            | :heavy_check_mark:                                                               | The context to use for the request.                                              |
| `request`                                                                        | [operations.ListTriggersRequest](../../models/operations/listtriggersrequest.md) | :heavy_check_mark:                                                               | The request object to use for the request.                                       |
| `opts`                                                                           | [][operations.Option](../../models/operations/option.md)                         | :heavy_minus_sign:                                                               | The options for this request.                                                    |


### Response

**[*operations.ListTriggersResponse](../../models/operations/listtriggersresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    res, err := s.Orchestration.V1.CreateTrigger(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateTriggerResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                        | Type                                                             | Required                                                         | Description                                                      |
| ---------------------------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------- | ---------------------------------------------------------------- |
| `ctx`                                                            | [context.Context](https://pkg.go.dev/context#Context)            | :heavy_check_mark:                                               | The context to use for the request.                              |
| `request`                                                        | [components.TriggerData](../../models/components/triggerdata.md) | :heavy_check_mark:                                               | The request object to use for the request.                       |
| `opts`                                                           | [][operations.Option](../../models/operations/option.md)         | :heavy_minus_sign:                                               | The options for this request.                                    |


### Response

**[*operations.CreateTriggerResponse](../../models/operations/createtriggerresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.ReadTriggerRequest{
        TriggerID: "<value>",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.ReadTrigger(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.ReadTriggerResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.ReadTriggerRequest](../../models/operations/readtriggerrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |


### Response

**[*operations.ReadTriggerResponse](../../models/operations/readtriggerresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.DeleteTriggerRequest{
        TriggerID: "<value>",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.DeleteTrigger(ctx, request)
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
| `request`                                                                          | [operations.DeleteTriggerRequest](../../models/operations/deletetriggerrequest.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |
| `opts`                                                                             | [][operations.Option](../../models/operations/option.md)                           | :heavy_minus_sign:                                                                 | The options for this request.                                                      |


### Response

**[*operations.DeleteTriggerResponse](../../models/operations/deletetriggerresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.ListTriggersOccurrencesRequest{
        TriggerID: "<value>",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.ListTriggersOccurrences(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListTriggersOccurrencesResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                              | Type                                                                                                   | Required                                                                                               | Description                                                                                            |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                                  | :heavy_check_mark:                                                                                     | The context to use for the request.                                                                    |
| `request`                                                                                              | [operations.ListTriggersOccurrencesRequest](../../models/operations/listtriggersoccurrencesrequest.md) | :heavy_check_mark:                                                                                     | The request object to use for the request.                                                             |
| `opts`                                                                                                 | [][operations.Option](../../models/operations/option.md)                                               | :heavy_minus_sign:                                                                                     | The options for this request.                                                                          |


### Response

**[*operations.ListTriggersOccurrencesResponse](../../models/operations/listtriggersoccurrencesresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## ListWorkflows

List registered workflows

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
    res, err := s.Orchestration.V1.ListWorkflows(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListWorkflowsResponse != nil {
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

**[*operations.ListWorkflowsResponse](../../models/operations/listworkflowsresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    res, err := s.Orchestration.V1.CreateWorkflow(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateWorkflowResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                            | Type                                                                                 | Required                                                                             | Description                                                                          |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| `ctx`                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                | :heavy_check_mark:                                                                   | The context to use for the request.                                                  |
| `request`                                                                            | [components.CreateWorkflowRequest](../../models/components/createworkflowrequest.md) | :heavy_check_mark:                                                                   | The request object to use for the request.                                           |
| `opts`                                                                               | [][operations.Option](../../models/operations/option.md)                             | :heavy_minus_sign:                                                                   | The options for this request.                                                        |


### Response

**[*operations.CreateWorkflowResponse](../../models/operations/createworkflowresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.GetWorkflowRequest{
        FlowID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.GetWorkflow(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetWorkflowResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.GetWorkflowRequest](../../models/operations/getworkflowrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |


### Response

**[*operations.GetWorkflowResponse](../../models/operations/getworkflowresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.DeleteWorkflowRequest{
        FlowID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.DeleteWorkflow(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                            | Type                                                                                 | Required                                                                             | Description                                                                          |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| `ctx`                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                | :heavy_check_mark:                                                                   | The context to use for the request.                                                  |
| `request`                                                                            | [operations.DeleteWorkflowRequest](../../models/operations/deleteworkflowrequest.md) | :heavy_check_mark:                                                                   | The request object to use for the request.                                           |
| `opts`                                                                               | [][operations.Option](../../models/operations/option.md)                             | :heavy_minus_sign:                                                                   | The options for this request.                                                        |


### Response

**[*operations.DeleteWorkflowResponse](../../models/operations/deleteworkflowresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.RunWorkflowRequest{
        WorkflowID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.RunWorkflow(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.RunWorkflowResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.RunWorkflowRequest](../../models/operations/runworkflowrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |


### Response

**[*operations.RunWorkflowResponse](../../models/operations/runworkflowresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.ListInstancesRequest{
        WorkflowID: openapi.String("xxx"),
        Running: openapi.Bool(true),
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.ListInstances(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListRunsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                          | Type                                                                               | Required                                                                           | Description                                                                        |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `ctx`                                                                              | [context.Context](https://pkg.go.dev/context#Context)                              | :heavy_check_mark:                                                                 | The context to use for the request.                                                |
| `request`                                                                          | [operations.ListInstancesRequest](../../models/operations/listinstancesrequest.md) | :heavy_check_mark:                                                                 | The request object to use for the request.                                         |
| `opts`                                                                             | [][operations.Option](../../models/operations/option.md)                           | :heavy_minus_sign:                                                                 | The options for this request.                                                      |


### Response

**[*operations.ListInstancesResponse](../../models/operations/listinstancesresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.GetInstanceRequest{
        InstanceID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.GetInstance(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetWorkflowInstanceResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.GetInstanceRequest](../../models/operations/getinstancerequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |


### Response

**[*operations.GetInstanceResponse](../../models/operations/getinstanceresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.SendEventRequest{
        InstanceID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.SendEvent(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                  | Type                                                                       | Required                                                                   | Description                                                                |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `ctx`                                                                      | [context.Context](https://pkg.go.dev/context#Context)                      | :heavy_check_mark:                                                         | The context to use for the request.                                        |
| `request`                                                                  | [operations.SendEventRequest](../../models/operations/sendeventrequest.md) | :heavy_check_mark:                                                         | The request object to use for the request.                                 |
| `opts`                                                                     | [][operations.Option](../../models/operations/option.md)                   | :heavy_minus_sign:                                                         | The options for this request.                                              |


### Response

**[*operations.SendEventResponse](../../models/operations/sendeventresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.CancelEventRequest{
        InstanceID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.CancelEvent(ctx, request)
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
| `request`                                                                      | [operations.CancelEventRequest](../../models/operations/canceleventrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |


### Response

**[*operations.CancelEventResponse](../../models/operations/canceleventresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.GetInstanceHistoryRequest{
        InstanceID: "xxx",
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.GetInstanceHistory(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetWorkflowInstanceHistoryResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                    | Type                                                                                         | Required                                                                                     | Description                                                                                  |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `ctx`                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                        | :heavy_check_mark:                                                                           | The context to use for the request.                                                          |
| `request`                                                                                    | [operations.GetInstanceHistoryRequest](../../models/operations/getinstancehistoryrequest.md) | :heavy_check_mark:                                                                           | The request object to use for the request.                                                   |
| `opts`                                                                                       | [][operations.Option](../../models/operations/option.md)                                     | :heavy_minus_sign:                                                                           | The options for this request.                                                                |


### Response

**[*operations.GetInstanceHistoryResponse](../../models/operations/getinstancehistoryresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
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
    request := operations.GetInstanceStageHistoryRequest{
        InstanceID: "xxx",
        Number: 0,
    }
    ctx := context.Background()
    res, err := s.Orchestration.V1.GetInstanceStageHistory(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetWorkflowInstanceHistoryStageResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                              | Type                                                                                                   | Required                                                                                               | Description                                                                                            |
| ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                  | [context.Context](https://pkg.go.dev/context#Context)                                                  | :heavy_check_mark:                                                                                     | The context to use for the request.                                                                    |
| `request`                                                                                              | [operations.GetInstanceStageHistoryRequest](../../models/operations/getinstancestagehistoryrequest.md) | :heavy_check_mark:                                                                                     | The request object to use for the request.                                                             |
| `opts`                                                                                                 | [][operations.Option](../../models/operations/option.md)                                               | :heavy_minus_sign:                                                                                     | The options for this request.                                                                          |


### Response

**[*operations.GetInstanceStageHistoryResponse](../../models/operations/getinstancestagehistoryresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.Error    | default            | application/json   |
| sdkerrors.SDKError | 4xx-5xx            | */*                |
