# V2TriggerTest

Result of evaluating a trigger against a sample event, without running the workflow


## Fields

| Field                                                                   | Type                                                                    | Required                                                                | Description                                                             |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `Filter`                                                                | [*components.Filter](../../models/components/filter.md)                 | :heavy_minus_sign:                                                      | How the trigger's filter evaluated against the sample event             |
| `Variables`                                                             | map[string][components.Variables](../../models/components/variables.md) | :heavy_minus_sign:                                                      | The variables the trigger would build from the sample event             |