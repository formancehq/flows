# Triggers

Triggers allow you to automatically start a workflow in response to events. When an event matches a trigger's criteria, the associated workflow is executed with variables extracted from the event payload.

## Trigger Structure

| Field        | Type              | Required | Description                                      |
|--------------|-------------------|----------|--------------------------------------------------|
| `event`      | string            | Yes      | The event type to listen for                     |
| `workflowID` | string            | Yes      | The ID of the workflow to execute                |
| `name`       | string            | No       | A human-readable name for the trigger            |
| `filter`     | string            | No       | An expression to filter events                   |
| `vars`       | map[string]string | No       | Expressions to extract variables from the event  |

## Filter

The `filter` field is an optional expression that determines whether the trigger should fire. It uses the [expr-lang](https://github.com/expr-lang/expr) syntax and must evaluate to a boolean.

The event payload is accessible via the `event` variable.

### Filter Examples

```
# Simple equality check
event.type == "payment"

# Nested field access
event.transaction.status == "completed"

# Numeric comparisons
event.amount > 1000
event.transaction.amount >= 500.50

# Combining conditions
event.type == "transfer" && event.amount > 100

# String matching
event.metadata.provider == "stripe"
```

### Filter Behavior

- If `filter` is not provided, the trigger fires for all events matching the `event` type.
- If `filter` evaluates to `true`, the workflow is executed.
- If `filter` evaluates to `false` or a non-boolean value, the workflow is not executed.

## Vars

The `vars` field is an optional map that extracts values from the event payload and passes them as input variables to the workflow.

- **Keys**: Variable names that will be available in the workflow
- **Values**: [expr-lang](https://github.com/expr-lang/expr) expressions evaluated against the event

All extracted values are converted to strings before being passed to the workflow.

### Vars Examples

```json
{
  "accountId": "event.account.id",
  "amount": "event.transaction.amount",
  "currency": "event.transaction.currency",
  "provider": "event.metadata.psp"
}
```

### Expression Features

#### Nested Field Access

```json
{
  "role": "event.user.profile.role"
}
```

#### String Concatenation

```json
{
  "reference": "event.prefix + \"-\" + event.id"
}
```

#### Accessing Array Elements

```json
{
  "firstItem": "event.items[0].name"
}
```

### The `link()` Function

Triggers support a special `link()` function that follows HATEOAS links embedded in the event payload. This allows you to fetch related resources during variable evaluation.

```json
{
  "sourceAccountRole": "link(event, \"source_account\").role"
}
```

The `link()` function:
1. Takes the event object and a link relation name
2. Finds the matching link in the event's `links` array
3. Performs an HTTP GET request to fetch the linked resource
4. Returns the resource data for further field access

#### Link Structure in Events

For `link()` to work, your event must contain a `links` array:

```json
{
  "id": "evt_123",
  "links": [
    {
      "name": "source_account",
      "uri": "https://api.example.com/accounts/acc_456"
    }
  ]
}
```

## Complete Example

### Trigger Definition

```json
{
  "name": "high-value-transfer-handler",
  "event": "ledger.committed.transactions",
  "workflowID": "wf_review_large_transfers",
  "filter": "event.transaction.amount > 10000 && event.transaction.type == \"transfer\"",
  "vars": {
    "transactionId": "event.transaction.id",
    "amount": "event.transaction.amount",
    "sourceAccount": "event.transaction.source",
    "destinationAccount": "event.transaction.destination",
    "initiatedBy": "event.metadata.user_id"
  }
}
```

### Incoming Event

```json
{
  "transaction": {
    "id": "tx_789",
    "type": "transfer",
    "amount": 25000,
    "source": "acc_001",
    "destination": "acc_002"
  },
  "metadata": {
    "user_id": "user_123"
  }
}
```

### Resulting Workflow Variables

The workflow will be started with:

```json
{
  "transactionId": "tx_789",
  "amount": "25000",
  "sourceAccount": "acc_001",
  "destinationAccount": "acc_002",
  "initiatedBy": "user_123"
}
```

## Testing Triggers

You can test a trigger's filter and variable expressions without actually firing the workflow using the test endpoint:

```
POST /triggers/{triggerId}/test
```

With a sample event payload in the request body. The response will show:
- Whether the filter matched
- The evaluated variable values
- Any errors in the expressions

## Error Handling

- If a `filter` expression fails to compile, the trigger creation will be rejected with an `ExprCompilationError`.
- If a `vars` expression fails to compile, the trigger creation will be rejected.
- If variable evaluation fails at runtime, the trigger occurrence is recorded with an error and the workflow is not started.
