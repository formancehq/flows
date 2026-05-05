package activities

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestCreateTransferInitiationSendsConnectorID(t *testing.T) {
	var transferInitiationRequest map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		require.Equal(t, "/api/payments/transfer-initiations", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&transferInitiationRequest))
		_, _ = w.Write([]byte(`{"data":{"id":"ti1","connectorID":"conn_routable","amount":100,"initialAmount":100,"asset":"USD/2","description":"routable PAYOUT","destinationAccountID":"dest","reference":"ref","status":"PROCESSING","type":"PAYOUT"}}`))
	}))
	t.Cleanup(server.Close)

	client := sdk.New(sdk.WithServerURL(server.URL), sdk.WithClient(server.Client()))
	activities := New(client)

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(activities.CreateTransferInitiation)

	_, err := env.ExecuteActivity(activities.CreateTransferInitiation, CreateTransferInitiationRequest{
		Amount:      big.NewInt(100),
		Asset:       pointer.For("USD/2"),
		Provider:    pointer.For("routable"),
		Type:        "PAYOUT",
		Destination: pointer.For("dest"),
		ConnectorID: pointer.For("conn_routable"),
	})
	require.NoError(t, err)
	require.Equal(t, "conn_routable", transferInitiationRequest["connectorID"])
}

func TestCreateTransferInitiationRequiresConnectorID(t *testing.T) {
	client := sdk.New()
	activities := New(client)

	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	env.RegisterActivity(activities.CreateTransferInitiation)

	_, err := env.ExecuteActivity(activities.CreateTransferInitiation, CreateTransferInitiationRequest{
		Amount:      big.NewInt(100),
		Asset:       pointer.For("USD/2"),
		Provider:    pointer.For("routable"),
		Type:        "PAYOUT",
		Destination: pointer.For("dest"),
	})
	require.ErrorContains(t, err, "connectorID is required")
}
