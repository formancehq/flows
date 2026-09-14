# V2CreateWorkflowRequest

The stages a workflow runs, in order


## Fields

| Field                                               | Type                                                | Required                                            | Description                                         |
| --------------------------------------------------- | --------------------------------------------------- | --------------------------------------------------- | --------------------------------------------------- |
| `Name`                                              | **string*                                           | :heavy_minus_sign:                                  | Human-readable name for the workflow                |
| `Stages`                                            | []map[string]*any*                                  | :heavy_check_mark:                                  | The stages executed in order when the workflow runs |