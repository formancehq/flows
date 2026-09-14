package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk/producthttp"
	flowsclient "openapi"
	"openapi/models/components"
	"openapi/models/operations"
)

const generatedClientServerURL = "https://product.invalid"

func executeV2(ctx context.Context, request sdk.ExecuteRequest, host sdk.Host) error {
	command, ok := commandByID(request.CommandID)
	if !ok {
		return invalid("unknown command %q", request.CommandID)
	}
	if err := sdk.ValidateExecuteRequest(command, request); err != nil {
		return invalid("invalid execution request: %s", err)
	}
	if err := sdk.ValidateTargetSelection(command.Target, request.Target); err != nil {
		return invalid("invalid target: %s", err)
	}
	flags, err := collectFlags(command, request.Flags)
	if err != nil {
		return err
	}

	switch request.CommandID {
	case "flows.v2.triggers.list":
		return listTriggers(ctx, host, command, request, flags)
	case "flows.v2.triggers.show":
		client, err := generatedV2(host, command.Operations[0])
		if err != nil {
			return err
		}
		response, err := client.ReadTrigger(ctx, operations.V2ReadTriggerRequest{TriggerID: request.Arguments[0]})
		if err != nil {
			return generatedError("v2ReadTrigger", err)
		}
		body, err := requireBody("v2ReadTrigger", response.V2ReadTriggerResponse)
		if err != nil {
			return err
		}
		return emitJSON(host, command.ID, sdk.ResultObject, body.Data, nil)
	case "flows.v2.triggers.create":
		body := &components.V2TriggerData{Event: request.Arguments[0], WorkflowID: request.Arguments[1]}
		body.Name = optionalString(flags["name"])
		body.Version = optionalString(flags["version"])
		body.Filter = optionalString(flags["filter"])
		if raw := flags["vars"]; raw != "" {
			body.Vars, err = decodeJSONObjectLossless(raw)
			if err != nil {
				return invalid("vars must be a JSON object")
			}
		}
		client, err := generatedV2(host, command.Operations[0])
		if err != nil {
			return err
		}
		response, err := client.CreateTrigger(ctx, body)
		if err != nil {
			return generatedError("v2CreateTrigger", err)
		}
		result, err := requireBody("v2CreateTrigger", response.V2CreateTriggerResponse)
		if err != nil {
			return err
		}
		return emitJSON(host, command.ID, sdk.ResultObject, result.Data, nil)
	case "flows.v2.triggers.delete":
		client, err := generatedV2(host, command.Operations[0])
		if err != nil {
			return err
		}
		if _, err := client.DeleteTrigger(ctx, operations.V2DeleteTriggerRequest{TriggerID: request.Arguments[0]}); err != nil {
			return generatedError("v2DeleteTrigger", err)
		}
		return emitJSON(host, command.ID, sdk.ResultEmpty, map[string]any{}, nil)
	case "flows.v2.triggers.test":
		event, err := decodeJSONObjectLossless(request.Arguments[1])
		if err != nil {
			return invalid("event must be a JSON object")
		}
		client, err := generatedV2(host, command.Operations[0])
		if err != nil {
			return err
		}
		response, err := client.TestTrigger(ctx, operations.TestTriggerRequest{TriggerID: request.Arguments[0], RequestBody: event})
		if err != nil {
			return generatedError("testTrigger", err)
		}
		body, err := requireBody("testTrigger", response.V2TestTriggerResponse)
		if err != nil {
			return err
		}
		return emitJSON(host, command.ID, sdk.ResultObject, body.Data, nil)
	case "flows.v2.triggers.occurrences.list":
		return listTriggerOccurrences(ctx, host, command, request, flags)
	case "flows.v2.workflows.list":
		return listWorkflows(ctx, host, command, request, flags)
	case "flows.v2.workflows.show":
		workflow, err := getWorkflow(ctx, host, command.Operations[0], request.Arguments[0])
		if err != nil {
			return err
		}
		return emitJSON(host, command.ID, sdk.ResultObject, workflow, nil)
	case "flows.v2.workflows.create":
		var definition components.V2CreateWorkflowRequest
		if err := json.Unmarshal([]byte(request.Arguments[0]), &definition); err != nil {
			return invalid("definition must be a JSON object")
		}
		if definition.Stages == nil {
			return invalid("definition must include stages")
		}
		client, err := generatedV2(host, command.Operations[0])
		if err != nil {
			return err
		}
		response, err := client.CreateWorkflow(ctx, &definition)
		if err != nil {
			return generatedError("v2CreateWorkflow", err)
		}
		body, err := requireBody("v2CreateWorkflow", response.V2CreateWorkflowResponse)
		if err != nil {
			return err
		}
		return emitJSON(host, command.ID, sdk.ResultObject, body.Data, nil)
	case "flows.v2.workflows.delete":
		client, err := generatedV2(host, command.Operations[0])
		if err != nil {
			return err
		}
		if _, err := client.DeleteWorkflow(ctx, operations.V2DeleteWorkflowRequest{FlowID: request.Arguments[0]}); err != nil {
			return generatedError("v2DeleteWorkflow", err)
		}
		return emitJSON(host, command.ID, sdk.ResultEmpty, map[string]any{}, nil)
	case "flows.v2.workflows.run":
		return runWorkflow(ctx, host, command, request, flags)
	case "flows.v2.instances.list":
		return listInstances(ctx, host, command, request, flags)
	case "flows.v2.instances.show":
		return showInstance(ctx, host, command, request.Arguments[0])
	case "flows.v2.instances.describe":
		return describeInstance(ctx, host, command, request.Arguments[0])
	case "flows.v2.instances.send-event":
		client, err := generatedV2(host, command.Operations[0])
		if err != nil {
			return err
		}
		if _, err := client.SendEvent(ctx, operations.V2SendEventRequest{InstanceID: request.Arguments[0], RequestBody: &operations.V2SendEventRequestBody{Name: request.Arguments[1]}}); err != nil {
			return generatedError("v2SendEvent", err)
		}
		return emitJSON(host, command.ID, sdk.ResultEmpty, map[string]any{}, nil)
	case "flows.v2.instances.stop":
		client, err := generatedV2(host, command.Operations[0])
		if err != nil {
			return err
		}
		if _, err := client.CancelEvent(ctx, operations.V2CancelEventRequest{InstanceID: request.Arguments[0]}); err != nil {
			return generatedError("v2CancelEvent", err)
		}
		return emitJSON(host, command.ID, sdk.ResultEmpty, map[string]any{}, nil)
	default:
		return invalid("command is not implemented")
	}
}

func runWorkflow(ctx context.Context, host sdk.Host, command sdk.Command, request sdk.ExecuteRequest, flags map[string]string) error {
	variables := map[string]string{}
	if raw := flags["variables"]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &variables); err != nil || variables == nil {
			return invalid("variables must be a JSON object with string values")
		}
	}
	client, err := generatedV2(host, command.Operations[0])
	if err != nil {
		return err
	}
	response, err := client.RunWorkflow(ctx, operations.V2RunWorkflowRequest{WorkflowID: request.Arguments[0], Wait: optionalBool(flags, "wait"), RequestBody: variables})
	if err != nil {
		return generatedError("v2RunWorkflow", err)
	}
	body, err := requireBody("v2RunWorkflow", response.V2RunWorkflowResponse)
	if err != nil {
		return err
	}
	workflow, err := getWorkflow(ctx, host, command.Operations[1], request.Arguments[0])
	if err != nil {
		return err
	}
	return emitJSON(host, command.ID, sdk.ResultObject, map[string]any{"instance": body.Data, "workflow": workflow}, nil)
}

func showInstance(ctx context.Context, host sdk.Host, command sdk.Command, instanceID string) error {
	client, err := generatedV2(host, command.Operations[0])
	if err != nil {
		return err
	}
	response, err := client.GetInstance(ctx, operations.V2GetInstanceRequest{InstanceID: instanceID})
	if err != nil {
		return generatedError("v2GetInstance", err)
	}
	body, err := requireBody("v2GetInstance", response.V2GetWorkflowInstanceResponse)
	if err != nil {
		return err
	}
	if body.Data.WorkflowID == "" {
		return fmt.Errorf("flows: instance response has no workflowID")
	}
	workflow, err := getWorkflow(ctx, host, command.Operations[1], body.Data.WorkflowID)
	if err != nil {
		return err
	}
	return emitJSON(host, command.ID, sdk.ResultObject, map[string]any{"instance": body.Data, "workflow": workflow}, nil)
}

func describeInstance(ctx context.Context, host sdk.Host, command sdk.Command, instanceID string) error {
	client, err := generatedV2(host, command.Operations[0])
	if err != nil {
		return err
	}
	response, err := client.GetInstanceHistory(ctx, operations.V2GetInstanceHistoryRequest{InstanceID: instanceID})
	if err != nil {
		return generatedError("v2GetInstanceHistory", err)
	}
	body, err := requireBody("v2GetInstanceHistory", response.V2GetWorkflowInstanceHistoryResponse)
	if err != nil {
		return err
	}
	stages := make([][]components.V2WorkflowInstanceHistoryStage, 0, len(body.Data))
	for index := range body.Data {
		if index+1 >= int(sdk.PortableMaxHostRequests) {
			return budget("instance history exceeds the stage request limit")
		}
		stageClient, err := generatedV2(host, command.Operations[1])
		if err != nil {
			return err
		}
		response, err := stageClient.GetInstanceStageHistory(ctx, operations.V2GetInstanceStageHistoryRequest{InstanceID: instanceID, Number: int64(index)})
		if err != nil {
			return generatedError("v2GetInstanceStageHistory", err)
		}
		stage, err := requireBody("v2GetInstanceStageHistory", response.V2GetWorkflowInstanceHistoryStageResponse)
		if err != nil {
			return err
		}
		stages = append(stages, stage.Data)
	}
	return emitJSON(host, command.ID, sdk.ResultObject, map[string]any{"history": body.Data, "stages": stages}, nil)
}

func getWorkflow(ctx context.Context, host sdk.Host, policy sdk.OperationPolicy, workflowID string) (components.V2Workflow, error) {
	client, err := generatedV2(host, policy)
	if err != nil {
		return components.V2Workflow{}, err
	}
	response, err := client.GetWorkflow(ctx, operations.V2GetWorkflowRequest{FlowID: workflowID})
	if err != nil {
		return components.V2Workflow{}, generatedError("v2GetWorkflow", err)
	}
	body, err := requireBody("v2GetWorkflow", response.V2GetWorkflowResponse)
	if err != nil {
		return components.V2Workflow{}, err
	}
	return body.Data, nil
}

type generatedPage[T any] struct {
	items   []T
	next    *string
	hasMore bool
}

func listTriggers(ctx context.Context, host sdk.Host, command sdk.Command, request sdk.ExecuteRequest, flags map[string]string) error {
	client, err := generatedV2(host, command.Operations[0])
	if err != nil {
		return err
	}
	return executePages(ctx, host, command.ID, request.Continuation, func(cursor *string) (generatedPage[components.V2Trigger], error) {
		input := operations.V2ListTriggersRequest{Cursor: cursor}
		if cursor == nil {
			input.Name = optionalString(flags["name"])
			input.PageSize = optionalInt64(flags["page-size"])
		}
		response, err := client.ListTriggers(ctx, input)
		if err != nil {
			return generatedPage[components.V2Trigger]{}, generatedError("v2ListTriggers", err)
		}
		body, err := requireBody("v2ListTriggers", response.V2ListTriggersResponse)
		if err != nil {
			return generatedPage[components.V2Trigger]{}, err
		}
		if err := validateGeneratedPage("v2ListTriggers", body.Cursor.PageSize, body.Cursor.Data, body.Cursor.HasMore, body.Cursor.Next); err != nil {
			return generatedPage[components.V2Trigger]{}, err
		}
		return generatedPage[components.V2Trigger]{items: body.Cursor.Data, next: body.Cursor.Next, hasMore: body.Cursor.HasMore}, nil
	})
}

func listTriggerOccurrences(ctx context.Context, host sdk.Host, command sdk.Command, request sdk.ExecuteRequest, flags map[string]string) error {
	client, err := generatedV2(host, command.Operations[0])
	if err != nil {
		return err
	}
	return executePages(ctx, host, command.ID, request.Continuation, func(cursor *string) (generatedPage[components.V2TriggerOccurrence], error) {
		input := operations.V2ListTriggersOccurrencesRequest{TriggerID: request.Arguments[0], Cursor: cursor}
		if cursor == nil {
			input.PageSize = optionalInt64(flags["page-size"])
		}
		response, err := client.ListTriggersOccurrences(ctx, input)
		if err != nil {
			return generatedPage[components.V2TriggerOccurrence]{}, generatedError("v2ListTriggersOccurrences", err)
		}
		body, err := requireBody("v2ListTriggersOccurrences", response.V2ListTriggersOccurrencesResponse)
		if err != nil {
			return generatedPage[components.V2TriggerOccurrence]{}, err
		}
		if err := validateGeneratedPage("v2ListTriggersOccurrences", body.Cursor.PageSize, body.Cursor.Data, body.Cursor.HasMore, body.Cursor.Next); err != nil {
			return generatedPage[components.V2TriggerOccurrence]{}, err
		}
		return generatedPage[components.V2TriggerOccurrence]{items: body.Cursor.Data, next: body.Cursor.Next, hasMore: body.Cursor.HasMore}, nil
	})
}

func listWorkflows(ctx context.Context, host sdk.Host, command sdk.Command, request sdk.ExecuteRequest, flags map[string]string) error {
	client, err := generatedV2(host, command.Operations[0])
	if err != nil {
		return err
	}
	return executePages(ctx, host, command.ID, request.Continuation, func(cursor *string) (generatedPage[components.V2Workflow], error) {
		input := operations.V2ListWorkflowsRequest{Cursor: cursor}
		if cursor == nil {
			input.PageSize = optionalInt64(flags["page-size"])
		}
		response, err := client.ListWorkflows(ctx, input)
		if err != nil {
			return generatedPage[components.V2Workflow]{}, generatedError("v2ListWorkflows", err)
		}
		body, err := requireBody("v2ListWorkflows", response.V2ListWorkflowsResponse)
		if err != nil {
			return generatedPage[components.V2Workflow]{}, err
		}
		if err := validateGeneratedPage("v2ListWorkflows", body.Cursor.PageSize, body.Cursor.Data, body.Cursor.HasMore, body.Cursor.Next); err != nil {
			return generatedPage[components.V2Workflow]{}, err
		}
		return generatedPage[components.V2Workflow]{items: body.Cursor.Data, next: body.Cursor.Next, hasMore: body.Cursor.HasMore}, nil
	})
}

func listInstances(ctx context.Context, host sdk.Host, command sdk.Command, request sdk.ExecuteRequest, flags map[string]string) error {
	client, err := generatedV2(host, command.Operations[0])
	if err != nil {
		return err
	}
	return executePages(ctx, host, command.ID, request.Continuation, func(cursor *string) (generatedPage[components.V2WorkflowInstance], error) {
		input := operations.V2ListInstancesRequest{Cursor: cursor}
		if cursor == nil {
			input.WorkflowID = optionalString(flags["workflow-id"])
			input.Running = optionalBool(flags, "running")
			input.PageSize = optionalInt64(flags["page-size"])
		}
		response, err := client.ListInstances(ctx, input)
		if err != nil {
			return generatedPage[components.V2WorkflowInstance]{}, generatedError("v2ListInstances", err)
		}
		body, err := requireBody("v2ListInstances", response.V2ListRunsResponse)
		if err != nil {
			return generatedPage[components.V2WorkflowInstance]{}, err
		}
		if err := validateGeneratedPage("v2ListInstances", body.Cursor.PageSize, body.Cursor.Data, body.Cursor.HasMore, body.Cursor.Next); err != nil {
			return generatedPage[components.V2WorkflowInstance]{}, err
		}
		return generatedPage[components.V2WorkflowInstance]{items: body.Cursor.Data, next: body.Cursor.Next, hasMore: body.Cursor.HasMore}, nil
	})
}

func executePages[T any](ctx context.Context, host sdk.Host, resultID string, control sdk.ContinuationControl, fetch func(*string) (generatedPage[T], error)) error {
	all := control.Mode == sdk.ContinuationAllPages
	max := uint32(1)
	if all {
		max = control.MaxPages
	}
	items := make([]T, 0)
	seen := map[string]struct{}{}
	var cursor *string
	for pageNumber := uint32(0); pageNumber < max; pageNumber++ {
		page, err := fetch(cursor)
		if err != nil {
			return err
		}
		if !all {
			return emitJSON(host, resultID, sdk.ResultCollection, page.items, &sdk.PageInfo{NextCursor: stringValue(page.next), HasMore: page.hasMore})
		}
		items = append(items, page.items...)
		encoded, err := json.Marshal(items)
		if err != nil {
			return fmt.Errorf("flows: encode collection: %w", err)
		}
		if uint32(len(items)) > control.MaxItems || uint64(len(encoded)) > control.MaxBytes {
			return budget("collection limit exceeded")
		}
		if !page.hasMore && stringValue(page.next) == "" {
			return emit(host, resultID, sdk.ResultCollection, encoded, nil)
		}
		next := stringValue(page.next)
		if next == "" {
			return fmt.Errorf("flows: pagination hasMore without next cursor")
		}
		if _, ok := seen[next]; ok {
			return fmt.Errorf("flows: repeated pagination cursor")
		}
		seen[next] = struct{}{}
		cursor = &next
	}
	return budget("page limit exceeded")
}

func generatedV2(host sdk.Host, policy sdk.OperationPolicy) (*flowsclient.V2, error) {
	client, err := producthttp.New(host, policy, "auth.stack")
	if err != nil {
		return nil, fmt.Errorf("flows: configure generated HTTP adapter: %w", err)
	}
	// Security and RetryConfig are intentionally absent. The host owns both.
	return flowsclient.New(flowsclient.WithServerURL(generatedClientServerURL), flowsclient.WithClient(client)).Orchestration.V2, nil
}

func validateGeneratedPage[T any](operation string, pageSize int64, items []T, hasMore bool, next *string) error {
	// The historical generated decoder does not enforce OpenAPI required fields.
	// Retain the spec's presence contract without decoding the response a second time.
	if pageSize < 1 || items == nil {
		return fmt.Errorf("flows: %s returned an incomplete cursor envelope", operation)
	}
	if hasMore && stringValue(next) == "" {
		return fmt.Errorf("flows: %s returned hasMore without a next cursor", operation)
	}
	if !hasMore && next != nil {
		return fmt.Errorf("flows: %s returned a next cursor without hasMore", operation)
	}
	return nil
}

func decodeJSONObjectLossless(raw string) (map[string]any, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}
	if value == nil {
		return nil, fmt.Errorf("decode JSON object: null is not an object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, fmt.Errorf("decode JSON object: %w", err)
	}
	return value, nil
}

func requireBody[T any](operation string, body *T) (*T, error) {
	if body == nil {
		return nil, fmt.Errorf("flows: %s returned no generated response body", operation)
	}
	return body, nil
}

func generatedError(operation string, err error) error {
	return fmt.Errorf("flows: %s: %w", operation, err)
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalInt64(value string) *int64 {
	if value == "" {
		return nil
	}
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return &parsed
}

func optionalBool(flags map[string]string, name string) *bool {
	value, ok := flags[name]
	if !ok {
		return nil
	}
	parsed, _ := strconv.ParseBool(value)
	return &parsed
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func commandByID(id string) (sdk.Command, bool) {
	for _, command := range Catalogue() {
		if command.ID == id {
			return command, true
		}
	}
	return sdk.Command{}, false
}

func collectFlags(command sdk.Command, occurrences []sdk.FlagOccurrence) (map[string]string, error) {
	declared := map[string]sdk.Flag{}
	for _, flag := range command.Flags {
		declared[flag.Name] = flag
	}
	out := map[string]string{}
	for _, occurrence := range occurrences {
		flag, ok := declared[occurrence.Name]
		if !ok {
			return nil, invalid("unknown flag --%s", occurrence.Name)
		}
		if _, exists := out[occurrence.Name]; exists {
			return nil, invalid("flag --%s is repeated", occurrence.Name)
		}
		if flag.Type == sdk.FlagBool {
			if _, err := strconv.ParseBool(occurrence.Value); err != nil {
				return nil, invalid("flag --%s must be boolean", occurrence.Name)
			}
		}
		if flag.Type == sdk.FlagInt32 {
			value, err := strconv.ParseInt(occurrence.Value, 10, 32)
			if err != nil || value < 1 {
				return nil, invalid("flag --%s must be positive", occurrence.Name)
			}
		}
		out[occurrence.Name] = occurrence.Value
	}
	return out, nil
}

func emitJSON(host sdk.Host, id string, shape sdk.ResultShape, value any, page *sdk.PageInfo) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("flows: encode result: %w", err)
	}
	return emit(host, id, shape, data, page)
}

func emit(host sdk.Host, id string, shape sdk.ResultShape, data []byte, page *sdk.PageInfo) error {
	return host.Emit(sdk.Event{Kind: sdk.EventResult, Result: &sdk.ResultEnvelope{OperationID: id, Shape: shape, MediaType: "application/json", Data: data, Page: page}})
}

func invalid(format string, args ...any) error {
	return sdk.Failure{Code: string(sdk.FailureInvalidArgument), Message: fmt.Sprintf("flows: "+format, args...)}
}

func budget(message string) error {
	return sdk.Failure{Code: string(sdk.FailureBudgetExhausted), Message: "flows: " + message}
}
