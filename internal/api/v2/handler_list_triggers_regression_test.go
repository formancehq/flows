package v2

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	sharedapi "github.com/formancehq/go-libs/v3/testing/api"
	"github.com/formancehq/orchestration/internal/api"
	"github.com/formancehq/orchestration/internal/triggers"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func insertNamedTriggerFixture(t *testing.T, db *bun.DB, workflowID, name string) triggers.Trigger {
	t.Helper()

	trigger := triggers.Trigger{
		TriggerData: triggers.TriggerData{
			Name:       name,
			Event:      "REGRESSION_EVENT",
			WorkflowID: workflowID,
		},
		ID:        uuid.NewString(),
		CreatedAt: time.Now().Round(time.Microsecond).UTC(),
	}
	_, err := db.NewInsert().Model(&trigger).Exec(t.Context())
	require.NoError(t, err)

	return trigger
}

func getTriggerCursor(t *testing.T, router *chi.Mux, rawQuery string) bunpaginate.Cursor[triggers.Trigger] {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/triggers?"+rawQuery, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Result().StatusCode)

	cursor := bunpaginate.Cursor[triggers.Trigger]{}
	sharedapi.ReadCursor(t, rec, &cursor)
	return cursor
}

func cursorNames(cursor *bunpaginate.Cursor[triggers.Trigger]) []string {
	names := make([]string, 0, len(cursor.Data))
	for _, trigger := range cursor.Data {
		names = append(names, trigger.Name)
	}
	slices.Sort(names)
	return names
}

func TestListTriggersNameFilterRegressionV2(t *testing.T) {
	test(t, func(router *chi.Mux, m api.Backend, db *bun.DB) {
		w := workflow.New(workflow.Config{})
		_, err := db.NewInsert().Model(&w).Exec(t.Context())
		require.NoError(t, err)

		names := []string{
			"payments-sync",
			"Payments-Refund",
			"100%_coverage",
			"o'brien-hook",
			"hook -- drop comment",
			"x'; DELETE FROM triggers; --",
			"plain-hook",
			"alphaXbeta",
			"alphaZZbeta",
			"o'brien -- archived",
			"boolean OR TRUE marker",
			"timing pg_sleep(1) marker",
		}
		byName := make(map[string]triggers.Trigger, len(names))
		for _, name := range names {
			byName[name] = insertNamedTriggerFixture(t, db, w.ID, name)
			deleted := insertNamedTriggerFixture(t, db, w.ID, name)
			require.NoError(t, m.DeleteTrigger(t.Context(), deleted.ID))
		}
		require.NoError(t, m.DeleteTrigger(t.Context(), byName["plain-hook"].ID))

		type testCase struct {
			name     string
			filter   string
			expected []string
		}
		testCases := []testCase{
			{name: "substring", filter: "ments-s", expected: []string{"payments-sync"}},
			{name: "case insensitive", filter: "PAYMENT", expected: []string{"Payments-Refund", "payments-sync"}},
			{name: "percent wildcard", filter: "100%", expected: []string{"100%_coverage"}},
			{name: "underscore wildcard", filter: "100_", expected: []string{"100%_coverage"}},
			{name: "apostrophe as data", filter: "o'brien", expected: []string{"o'brien -- archived", "o'brien-hook"}},
			{name: "comment-like as data", filter: "-- drop", expected: []string{"hook -- drop comment"}},
			{name: "SQL-looking name as data", filter: "DELETE FROM triggers", expected: []string{"x'; DELETE FROM triggers; --"}},
			{name: "deleted excluded", filter: "plain", expected: []string{}},
			{name: "no match", filter: "zzz", expected: []string{}},
			{name: "percent matches variable length", filter: "alpha%beta", expected: []string{"alphaXbeta", "alphaZZbeta"}},
			{name: "underscore matches one character", filter: "alpha_beta", expected: []string{"alphaXbeta"}},
			{name: "quote and comment markers as data", filter: "o'brien --", expected: []string{"o'brien -- archived"}},
			{name: "boolean text as data", filter: "OR TRUE", expected: []string{"boolean OR TRUE marker"}},
			{name: "timing text as data", filter: "pg_sleep(1)", expected: []string{"timing pg_sleep(1) marker"}},
			{name: "unmatched special text", filter: "missing' --", expected: []string{}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				query := triggers.ListTriggersQuery{
					PageSize: 100,
					Options:  triggers.ListTriggerParams{Name: tc.filter},
				}
				for delivery, rawQuery := range map[string]string{
					"direct": "name=" + url.QueryEscape(tc.filter),
					"cursor": "cursor=" + url.QueryEscape(bunpaginate.EncodeCursor(query)),
				} {
					t.Run(delivery, func(t *testing.T) {
						cursor := getTriggerCursor(t, router, rawQuery)
						require.Equal(t, tc.expected, cursorNames(&cursor))
					})
				}
			})
		}

		// No filter: everything except the deleted trigger.
		cursor := getTriggerCursor(t, router, "")
		expected := slices.DeleteFunc(slices.Clone(names), func(name string) bool { return name == "plain-hook" })
		slices.Sort(expected)
		require.Equal(t, expected, cursorNames(&cursor))

		// Cursor-based extraction through the HTTP route: page 1 with
		// pageSize=2, then follow the returned ?cursor= link.
		for i := 0; i < 3; i++ {
			insertNamedTriggerFixture(t, db, w.ID, "paged-v2-"+string(rune('0'+i)))
		}

		first := getTriggerCursor(t, router, "name=paged-v2&pageSize=2")
		require.Len(t, first.Data, 2)
		require.True(t, first.HasMore)
		require.NotEmpty(t, first.Next)

		collected := cursorNames(&first)
		next := first.Next
		for next != "" {
			page := getTriggerCursor(t, router, "cursor="+url.QueryEscape(next))
			collected = append(collected, cursorNames(&page)...)
			if !page.HasMore {
				break
			}
			next = page.Next
		}
		slices.Sort(collected)
		require.Equal(t, []string{"paged-v2-0", "paged-v2-1", "paged-v2-2"}, collected)
	})
}
