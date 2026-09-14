# StageSendDestination

Where a send stage puts the funds


## Fields

| Field                                                                                             | Type                                                                                              | Required                                                                                          | Description                                                                                       |
| ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `Wallet`                                                                                          | [*components.StageSendSourceWallet](../../models/components/stagesendsourcewallet.md)             | :heavy_minus_sign:                                                                                | Send the funds to a wallet                                                                        |
| `Account`                                                                                         | [*components.StageSendSourceAccount](../../models/components/stagesendsourceaccount.md)           | :heavy_minus_sign:                                                                                | Send the funds to a ledger account                                                                |
| `Payment`                                                                                         | [*components.StageSendDestinationPayment](../../models/components/stagesenddestinationpayment.md) | :heavy_minus_sign:                                                                                | Send the funds to a payment                                                                       |