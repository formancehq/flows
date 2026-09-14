# V2StageStatus


## Fields

| Field                                                     | Type                                                      | Required                                                  | Description                                               |
| --------------------------------------------------------- | --------------------------------------------------------- | --------------------------------------------------------- | --------------------------------------------------------- |
| `Stage`                                                   | *float64*                                                 | :heavy_check_mark:                                        | Zero-based position of the stage within the workflow      |
| `InstanceID`                                              | *string*                                                  | :heavy_check_mark:                                        | Identifier of the workflow instance this stage belongs to |
| `StartedAt`                                               | [time.Time](https://pkg.go.dev/time#Time)                 | :heavy_check_mark:                                        | When the stage started                                    |
| `TerminatedAt`                                            | [*time.Time](https://pkg.go.dev/time#Time)                | :heavy_minus_sign:                                        | When the stage finished, absent while it is still running |
| `Error`                                                   | **string*                                                 | :heavy_minus_sign:                                        | Why the stage failed, absent when it succeeded            |