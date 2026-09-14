# V2StageSendDestination

Where a send stage puts the funds


## Fields

| Field                                                                                                 | Type                                                                                                  | Required                                                                                              | Description                                                                                           |
| ----------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| `Wallet`                                                                                              | [*components.V2StageSendSourceWallet](../../models/components/v2stagesendsourcewallet.md)             | :heavy_minus_sign:                                                                                    | Send the funds to a wallet                                                                            |
| `Account`                                                                                             | [*components.V2StageSendSourceAccount](../../models/components/v2stagesendsourceaccount.md)           | :heavy_minus_sign:                                                                                    | Send the funds to a ledger account                                                                    |
| `Payment`                                                                                             | [*components.V2StageSendDestinationPayment](../../models/components/v2stagesenddestinationpayment.md) | :heavy_minus_sign:                                                                                    | Send the funds to a payment                                                                           |