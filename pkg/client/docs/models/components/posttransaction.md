# PostTransaction


## Fields

| Field                                                      | Type                                                       | Required                                                   | Description                                                | Example                                                    |
| ---------------------------------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------- |
| `Timestamp`                                                | [*time.Time](https://pkg.go.dev/time#Time)                 | :heavy_minus_sign:                                         | N/A                                                        |                                                            |
| `Postings`                                                 | [][components.Posting](../../models/components/posting.md) | :heavy_minus_sign:                                         | N/A                                                        |                                                            |
| `Script`                                                   | [*components.Script](../../models/components/script.md)    | :heavy_minus_sign:                                         | N/A                                                        |                                                            |
| `Reference`                                                | **string*                                                  | :heavy_minus_sign:                                         | N/A                                                        | ref:001                                                    |
| `Metadata`                                                 | map[string]*string*                                        | :heavy_check_mark:                                         | N/A                                                        | {<br/>"admin": "true"<br/>}                                |