package component

import (
	"testing"

	pb "github.com/formancehq/fctl-v2-poc/pkg/plugin/protocol/componentbridgev1alpha1"
	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk/portable/component"
	"github.com/formancehq/orchestration/plugins/fctl/core"
	"google.golang.org/protobuf/proto"
)

// TestOnlyTheTypedFailureCodeCrossesThePortableHostBoundary pins what a host
// actually observes when the product answers a Flows command with a non-2xx
// response. Flows' only entrypoint is the portable component, whose terminal
// frame carries a failure code and nothing else, so the typed
// product_http_error code is the whole host-visible contract: the adapter's
// message, its httpStatus details and its retryability verdict are discarded
// by the protocol. TestProductHTTPErrorsSurfaceTypedFromTheAdapter pins those
// three adapter-local values; this test pins that they do not cross.
func TestOnlyTheTypedFailureCodeCrossesThePortableHostBoundary(t *testing.T) {
	for _, status := range []int32{404, 409, 503} {
		lifecycle, err := component.New(component.Providers{Command: core.Plugin{}}, Descriptor())
		if err != nil {
			t.Fatalf("configure component: %v", err)
		}
		const executionID = "failure-boundary"
		t.Cleanup(func() { lifecycle.Close(executionID) })

		started := lifecycle.Start(executionID, marshalEnvelope(t, executionID, pb.MessageKind_MESSAGE_KIND_START_EXECUTION, &pb.StartPayload{
			Start: &pb.StartPayload_Command{Command: &pb.CommandStart{
				CommandId: "flows.v2.workflows.show",
				Arguments: []string{"wf-1"},
				Target:    &pb.TargetCoordinates{OrganizationId: "org", StackId: "stack"},
				ServiceVersions: []*pb.ServiceVersion{
					{Service: pb.Service_SERVICE_FLOWS, Version: "2.0.0", Major: 2},
				},
				Continuation: &pb.ContinuationControl{Mode: pb.ContinuationMode_CONTINUATION_MODE_SINGLE_PAGE},
			}},
		}))
		if len(started) != 1 {
			t.Fatalf("HTTP %d: start frames = %d, want one host request", status, len(started))
		}
		var request pb.HostRequestPayload
		decodeFramePayload(t, started[0], pb.MessageKind_MESSAGE_KIND_HOST_REQUEST, &request)

		resumed := lifecycle.Resume(executionID, marshalEnvelope(t, executionID, pb.MessageKind_MESSAGE_KIND_HOST_RESPONSE, &pb.HostResponsePayload{
			CorrelationId: request.GetCorrelationId(),
			Response: &pb.HostResponsePayload_Product{Product: &pb.ProductResponse{
				Status: status, ContentType: "application/json", Body: []byte(`{"errorCode":"NOT_FOUND"}`),
			}},
		}))
		if len(resumed) != 1 {
			t.Fatalf("HTTP %d: resume frames = %d, want one termination", status, len(resumed))
		}
		var terminal pb.TerminationPayload
		decodeFramePayload(t, resumed[0], pb.MessageKind_MESSAGE_KIND_TERMINATION, &terminal)

		failure := terminal.GetFailure()
		if failure == nil {
			t.Fatalf("HTTP %d terminated with %#v, want a failure", status, &terminal)
		}
		if failure.GetCode() != string(sdk.FailureProductHTTPError) {
			t.Errorf("HTTP %d crossed the boundary as %q, want %q",
				status, failure.GetCode(), sdk.FailureProductHTTPError)
		}
		// The portable terminal frame has no field for these, so a host cannot
		// build a retry policy on them today. Pinned so a future SDK frame that
		// widens the boundary fails here and forces the README to be updated.
		if failure.GetMessage() != "" || len(failure.GetDetails()) != 0 || failure.GetRetryable() {
			t.Errorf("HTTP %d crossed the boundary with message=%q details=%q retryable=%t; the portable terminal frame carries only a code",
				status, failure.GetMessage(), failure.GetDetails(), failure.GetRetryable())
		}
	}
}

func marshalEnvelope(t *testing.T, executionID string, kind pb.MessageKind, payload proto.Message) []byte {
	t.Helper()
	encodedPayload, err := proto.MarshalOptions{Deterministic: true}.Marshal(payload)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(&pb.PluginEnvelope{
		ProtocolMajor: pb.ProtocolMajor,
		FacetKind:     pb.FacetKind_FACET_KIND_COMMAND_PROVIDER,
		MessageKind:   kind,
		ExecutionId:   executionID,
		Payload:       encodedPayload,
	})
	if err != nil {
		t.Fatalf("encode envelope: %v", err)
	}
	return encoded
}

func decodeFramePayload(t *testing.T, frame []byte, kind pb.MessageKind, payload proto.Message) {
	t.Helper()
	var envelope pb.PluginEnvelope
	if err := proto.Unmarshal(frame, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.GetMessageKind() != kind {
		t.Fatalf("frame kind = %s, want %s", envelope.GetMessageKind(), kind)
	}
	if err := proto.Unmarshal(envelope.GetPayload(), payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
}
