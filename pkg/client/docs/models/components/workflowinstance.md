# WorkflowInstance

One run of a workflow, tracking its per-stage progress


## Fields

| Field                                                              | Type                                                               | Required                                                           | Description                                                        |
| ------------------------------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------------------------------ |
| `WorkflowID`                                                       | *string*                                                           | :heavy_check_mark:                                                 | Identifier of the workflow this run belongs to                     |
| `ID`                                                               | *string*                                                           | :heavy_check_mark:                                                 | Unique identifier of the run                                       |
| `CreatedAt`                                                        | [time.Time](https://pkg.go.dev/time#Time)                          | :heavy_check_mark:                                                 | When the run was started                                           |
| `UpdatedAt`                                                        | [time.Time](https://pkg.go.dev/time#Time)                          | :heavy_check_mark:                                                 | When the run was last updated                                      |
| `Status`                                                           | [][components.StageStatus](../../models/components/stagestatus.md) | :heavy_minus_sign:                                                 | Per-stage progress of the run                                      |
| `Terminated`                                                       | *bool*                                                             | :heavy_check_mark:                                                 | Whether the run has finished, successfully or not                  |
| `TerminatedAt`                                                     | [*time.Time](https://pkg.go.dev/time#Time)                         | :heavy_minus_sign:                                                 | When the run finished, absent while it is still running            |
| `Error`                                                            | **string*                                                          | :heavy_minus_sign:                                                 | Why the run failed, absent when it succeeded                       |
| `Workflow`                                                         | [*components.Workflow](../../models/components/workflow.md)        | :heavy_minus_sign:                                                 | A workflow definition and the stages it runs                       |