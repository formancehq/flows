# V2WorkflowInstanceHistory


## Fields

| Field                                                         | Type                                                          | Required                                                      | Description                                                   |
| ------------------------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------- | ------------------------------------------------------------- |
| `Name`                                                        | *string*                                                      | :heavy_check_mark:                                            | Name of the stage this history entry records                  |
| `Input`                                                       | [components.V2Stage](../../models/components/v2stage.md)      | :heavy_check_mark:                                            | One step of a workflow, whose shape depends on the stage type |
| `Error`                                                       | **string*                                                     | :heavy_minus_sign:                                            | Why the stage failed, absent when it succeeded                |
| `Terminated`                                                  | *bool*                                                        | :heavy_check_mark:                                            | Whether the stage has finished                                |
| `StartedAt`                                                   | [time.Time](https://pkg.go.dev/time#Time)                     | :heavy_check_mark:                                            | When the stage started                                        |
| `TerminatedAt`                                                | [*time.Time](https://pkg.go.dev/time#Time)                    | :heavy_minus_sign:                                            | When the stage finished, absent while it is still running     |