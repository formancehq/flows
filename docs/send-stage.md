# Send Stage

The `send` stage moves funds between accounts, wallets, and payment service providers (PSPs). It supports various source and destination combinations with customizable ledger transaction behavior.

## Source Types

### Ledger Account Source

Send funds from a ledger account.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | Yes | - | The ledger account address |
| `ledger` | string | Yes | - | The ledger name |
| `throughAccount` | string | No | `"world"` | Intermediate account for cross-ledger or payment flows |
| `allowOverdraft` | bool | No | `false` | Allow unbounded overdraft on the source account |

```yaml
source:
  account:
    id: "users:123"
    ledger: "main"
    throughAccount: "liabilities:pending"  # Optional: customize bridge account
    allowOverdraft: true                   # Optional: allow negative balance
```

### Wallet Source

Send funds from a wallet.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes* | The wallet ID |
| `name` | string | Yes* | The wallet name (alternative to ID) |
| `balance` | string | No | Specific balance to debit (default: main) |

*Either `id` or `name` is required.

```yaml
source:
  wallet:
    id: "${walletID}"
    balance: "main"
```

### Payment Source

Send funds from an incoming payment (payin). The payment is first ingested into a ledger before being transferred to the destination.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | Yes | - | The payment ID from the Payments service |
| `ledger` | string | No | `"orchestration-000-internal"` | Ledger for payment ingestion |
| `holdingAccount` | string | No | `"payment:{id}"` | Account where payment funds are held |
| `throughAccount` | string | No | `"world"` | Source account for the ingestion transaction |
| `allowOverdraft` | bool | No | `false` | Allow unbounded overdraft on throughAccount |

```yaml
source:
  payment:
    id: "${paymentID}"
    # Optional: Use your own ledger instead of internal
    ledger: "main"
    holdingAccount: "assets:stripe:held"
    throughAccount: "assets:stripe:incoming"
    allowOverdraft: true
```

## Destination Types

### Ledger Account Destination

Send funds to a ledger account.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | Yes | - | The ledger account address |
| `ledger` | string | Yes | - | The ledger name |
| `throughAccount` | string | No | `"world"` | Intermediate account for cross-ledger or payment flows |
| `allowOverdraft` | bool | No | `false` | Allow unbounded overdraft on throughAccount (when receiving from different ledger) |

```yaml
destination:
  account:
    id: "merchants:456"
    ledger: "main"
    throughAccount: "assets:incoming"  # Optional
    allowOverdraft: true               # Optional
```

### Wallet Destination

Send funds to a wallet.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes* | The wallet ID |
| `name` | string | Yes* | The wallet name (alternative to ID) |
| `balance` | string | No | Specific balance to credit (default: main) |

*Either `id` or `name` is required.

```yaml
destination:
  wallet:
    id: "${walletID}"
```

### Payment Destination

Initiate a transfer or payout via a PSP connector.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `psp` | string | Yes | - | The PSP connector name (e.g., `"stripe"`, `"wise"`, `"modulr"`) |
| `type` | string | No | `"TRANSFER"` | Transfer type: `"TRANSFER"` or `"PAYOUT"` |
| `metadata` | string | No | - | Account metadata key containing the destination PSP account ID |
| `sourceAccount` | string | No | - | Explicit PSP source account ID for the transfer |
| `connectorID` | string | No | - | Specific connector ID (if multiple connectors of same type) |
| `waitingValidation` | bool | No | `false` | Whether to wait for manual validation |

```yaml
destination:
  payment:
    psp: "stripe"
    type: "PAYOUT"                        # TRANSFER or PAYOUT
    sourceAccount: "${sourceAccountID}"   # Optional: explicit source
    metadata: "stripeConnectID"           # Key in source account metadata
```

## Cross-Ledger Transfers

When source and destination are on different ledgers, the `throughAccount` field controls the intermediate account used on each side.

### Same Ledger (Direct Transfer)

```yaml
# Single transaction: users:123 → merchants:456
send:
  source:
    account:
      id: "users:123"
      ledger: "main"
  destination:
    account:
      id: "merchants:456"
      ledger: "main"  # Same ledger
```

### Different Ledgers (Bridge Transfer)

```yaml
# Two transactions:
# 1. ledger1: users:sender → bridge:outbound
# 2. ledger2: bridge:inbound → merchants:receiver
send:
  source:
    account:
      id: "users:sender"
      ledger: "ledger1"
      throughAccount: "bridge:outbound"
  destination:
    account:
      id: "merchants:receiver"
      ledger: "ledger2"
      throughAccount: "bridge:inbound"
      allowOverdraft: true  # bridge:inbound may need overdraft
```

## The `allowOverdraft` Field

By default, Formance Ledger requires accounts to have sufficient funds before debiting. The special `"world"` account has unbounded overdraft and is used by default for bridge transactions.

When using a custom `throughAccount` (not `"world"`), you may need to enable `allowOverdraft` to allow the account to go negative. This generates Numscript with the `allowing unbounded overdraft` clause.

### When Overdraft Applies

| Flow | Transaction | Overdraft Applied To |
|------|-------------|---------------------|
| Account → Payment | `source.ID → throughAccount` | `source.ID` |
| Payment → Account | `throughAccount → holdingAccount` | `throughAccount` |
| Account → Wallet (cross-ledger) | `source.ID → throughAccount` | `source.ID` |
| Wallet → Account (cross-ledger) | `throughAccount → destination.ID` | `throughAccount` |
| Account → Account (1st tx) | `source.ID → sourceThroughAccount` | `source.ID` |
| Account → Account (2nd tx) | `destThroughAccount → destination.ID` | `destThroughAccount` |

### Example: Liability Tracking for Payouts

```yaml
send:
  source:
    account:
      id: "users:${userID}"
      ledger: "main"
      throughAccount: "liabilities:payouts-pending"
      allowOverdraft: true  # User account may overdraft
  destination:
    payment:
      psp: "stripe"
      type: "PAYOUT"
```

This records the payout as: `users:{userID} → liabilities:payouts-pending` instead of `users:{userID} → world`.

## Payment Ingestion

When using a payment source, funds are first "ingested" into a ledger before being transferred. By default, this uses an internal orchestration ledger.

### Default Behavior

```
1. world → payment:{paymentID}  (on orchestration-000-internal)
2. payment:{paymentID} → world  (move out, with metadata)
3. world → destination          (on destination ledger)
```

### Custom Ingestion

You can customize where payments are ingested:

```yaml
source:
  payment:
    id: "${paymentID}"
    ledger: "main"                          # Your ledger
    holdingAccount: "assets:stripe:held"    # Custom holding account
    throughAccount: "assets:stripe:incoming"
    allowOverdraft: true
```

This produces:

```
1. assets:stripe:incoming → assets:stripe:held  (on main ledger)
2. assets:stripe:held → destination             (on main ledger)
```

## Supported PSP Connectors

The payment destination supports all PSP connectors configured in the Payments service:

- Stripe
- Wise
- Modulr
- Moneycorp
- Banking Circle
- Currency Cloud
- Mangopay
- And others...

The orchestration service delegates validation to the Payments service, allowing new connectors to work without code changes.

## Complete Examples

### Ledger to Payout with Tracking

```yaml
name: "payout-with-tracking"
stages:
- send:
    source:
      account:
        id: "users:${userID}"
        ledger: "main"
        throughAccount: "liabilities:payouts-pending"
        allowOverdraft: true
    destination:
      payment:
        psp: "${psp}"
        type: "PAYOUT"
        sourceAccount: "${sourceAccountID}"
    amount:
      amount: "${amount}"
      asset: "${asset}"
```

### Payment to Account with Custom Ingestion

```yaml
name: "payin-custom-ingestion"
stages:
- send:
    source:
      payment:
        id: "${paymentID}"
        ledger: "main"
        holdingAccount: "assets:stripe:pending"
        throughAccount: "assets:stripe:bridge"
        allowOverdraft: true
    destination:
      account:
        id: "revenue:${merchantID}"
        ledger: "main"
    amount:
      amount: "${amount}"
      asset: "${asset}"
```

### Cross-Ledger with Custom Bridge Accounts

```yaml
name: "cross-ledger-transfer"
stages:
- send:
    source:
      account:
        id: "users:${userID}"
        ledger: "users-ledger"
        throughAccount: "bridge:to-merchants"
    destination:
      account:
        id: "merchants:${merchantID}"
        ledger: "merchants-ledger"
        throughAccount: "bridge:from-users"
        allowOverdraft: true
    amount:
      amount: "${amount}"
      asset: "${asset}"
```
