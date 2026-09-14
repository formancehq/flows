# ActivityCreateTransaction

Arguments for the activity that writes a transaction to a ledger


## Fields

| Field                                                                     | Type                                                                      | Required                                                                  | Description                                                               |
| ------------------------------------------------------------------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `Ledger`                                                                  | **string*                                                                 | :heavy_minus_sign:                                                        | Name of the ledger to write the transaction to                            |
| `Data`                                                                    | [*components.PostTransaction](../../models/components/posttransaction.md) | :heavy_minus_sign:                                                        | A transaction to write to a ledger                                        |