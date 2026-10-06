package v2

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/orchestration/internal/api"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestWorkflowActivityMaxAttemptsCreateRead(t *testing.T) {
	for _, tc := range []struct {
		name, body, contentType string
		want                    int
	}{
		{"default-json", `{"name":"default","stages":[]}`, "application/json", 15},
		{"null-json", `{"stages":[],"activityMaxAttempts":null}`, "application/json", 15},
		{"default-yaml", "name: default\nstages: []\n", "text/vnd.yaml", 15},
		{"custom-json", `{"name":"custom","stages":[],"activityMaxAttempts":3}`, "application/json", 3},
		{"custom-yaml", "name: custom\nstages: []\nactivityMaxAttempts: 3\n", "text/vnd.yaml", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			test(t, func(router *chi.Mux, _ api.Backend, db *bun.DB) {
				createdResponse := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodPost, "/workflows", bytes.NewBufferString(tc.body))
				request.Header.Set("Content-Type", tc.contentType)
				router.ServeHTTP(createdResponse, request)
				require.Equal(t, http.StatusCreated, createdResponse.Code, createdResponse.Body.String())
				var created struct {
					Data workflow.Workflow `json:"data"`
				}
				require.NoError(t, json.Unmarshal(createdResponse.Body.Bytes(), &created))
				require.NotNil(t, created.Data.Config.ActivityMaxAttempts)
				require.Equal(t, tc.want, *created.Data.Config.ActivityMaxAttempts)

				var stored workflow.Workflow
				require.NoError(t, db.NewSelect().Model(&stored).Where("id = ?", created.Data.ID).Scan(t.Context()))
				require.NotNil(t, stored.Config.ActivityMaxAttempts)
				require.Equal(t, tc.want, *stored.Config.ActivityMaxAttempts)

				readResponse := httptest.NewRecorder()
				router.ServeHTTP(readResponse, httptest.NewRequest(http.MethodGet, "/workflows/"+created.Data.ID, nil))
				require.Equal(t, http.StatusOK, readResponse.Code, readResponse.Body.String())
				var read struct {
					Data workflow.Workflow `json:"data"`
				}
				require.NoError(t, json.Unmarshal(readResponse.Body.Bytes(), &read))
				require.Equal(t, created.Data.ID, read.Data.ID)
				require.NotNil(t, read.Data.Config.ActivityMaxAttempts)
				require.Equal(t, tc.want, *read.Data.Config.ActivityMaxAttempts)
			})
		})
	}
}

func TestWorkflowActivityMaxAttemptsRejectInvalid(t *testing.T) {
	for _, contentType := range []string{"application/json", "text/vnd.yaml"} {
		for _, value := range []string{"0", "-1", "2147483648"} {
			t.Run(contentType+"/"+value, func(t *testing.T) {
				test(t, func(router *chi.Mux, _ api.Backend, db *bun.DB) {
					body := `{"stages":[],"activityMaxAttempts":` + value + `}`
					if contentType == "text/vnd.yaml" {
						body = "stages: []\nactivityMaxAttempts: " + value + "\n"
					}
					request := httptest.NewRequest(http.MethodPost, "/workflows", bytes.NewBufferString(body))
					request.Header.Set("Content-Type", contentType)
					response := httptest.NewRecorder()
					router.ServeHTTP(response, request)
					require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
					require.Contains(t, response.Body.String(), "VALIDATION")
					require.Contains(t, response.Body.String(), "activityMaxAttempts")
					count, err := db.NewSelect().Model((*workflow.Workflow)(nil)).Count(t.Context())
					require.NoError(t, err)
					require.Zero(t, count)
				})
			})
		}
	}
}
