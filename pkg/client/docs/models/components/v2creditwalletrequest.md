# V2CreditWalletRequest


## Fields

| Field                                                          | Type                                                           | Required                                                       | Description                                                    |
| -------------------------------------------------------------- | -------------------------------------------------------------- | -------------------------------------------------------------- | -------------------------------------------------------------- |
| `Amount`                                                       | [components.V2Monetary](../../models/components/v2monetary.md) | :heavy_check_mark:                                             | N/A                                                            |
| `Metadata`                                                     | map[string]*string*                                            | :heavy_check_mark:                                             | Metadata associated with the wallet.                           |
| `Reference`                                                    | **string*                                                      | :heavy_minus_sign:                                             | N/A                                                            |
| `Sources`                                                      | [][components.V2Subject](../../models/components/v2subject.md) | :heavy_check_mark:                                             | N/A                                                            |
| `Balance`                                                      | **string*                                                      | :heavy_minus_sign:                                             | The balance to credit                                          |
| `Timestamp`                                                    | [*time.Time](https://pkg.go.dev/time#Time)                     | :heavy_minus_sign:                                             | N/A                                                            |