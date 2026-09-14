# V2Workflow

A workflow definition and the stages it runs


## Fields

| Field                                                                      | Type                                                                       | Required                                                                   | Description                                                                |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `Config`                                                                   | [components.V2WorkflowConfig](../../models/components/v2workflowconfig.md) | :heavy_check_mark:                                                         | The stages a workflow runs, in order                                       |
| `CreatedAt`                                                                | [time.Time](https://pkg.go.dev/time#Time)                                  | :heavy_check_mark:                                                         | When the workflow was created                                              |
| `UpdatedAt`                                                                | [time.Time](https://pkg.go.dev/time#Time)                                  | :heavy_check_mark:                                                         | When the workflow was last modified                                        |
| `ID`                                                                       | *string*                                                                   | :heavy_check_mark:                                                         | Unique identifier of the workflow                                          |