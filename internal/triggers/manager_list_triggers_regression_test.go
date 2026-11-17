package triggers

import (
	"slices"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func insertNamedTrigger(t *testing.T, db *bun.DB, workflowID, name string) Trigger {
	t.Helper()

	trigger := Trigger{
		TriggerData: TriggerData{
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

func triggerNames(cursor *bunpaginate.Cursor[Trigger]) []string {
	ret := make([]string, 0, len(cursor.Data))
	for _, trigger := range cursor.Data {
		ret = append(ret, trigger.Name)
	}
	slices.Sort(ret)
	return ret
}

// regressionFixtures returns a set of trigger names covering normal text,
// case variations, LIKE wildcard characters, apostrophes, comment-like text
// and SQL-looking text. All of them must be storable and searchable as plain
// data.
func regressionFixtures(t *testing.T, db *bun.DB, workflowID string) map[string]Trigger {
	names := []string{
		"payments-sync",
		"Payments-Refund",
		"100%_coverage",
		"o'brien-hook",
		"hook -- drop comment",
		"x'; DELETE FROM triggers; --",
		"plain-hook",
	}
	ret := make(map[string]Trigger, len(names))
	for _, name := range names {
		ret[name] = insertNamedTrigger(t, db, workflowID, name)
	}
	return ret
}

func TestListTriggersNameFilterRegression(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	w := insertNoOpWorkflow(t, db)
	manager := NewManager(db, NewExpressionEvaluator(nil))

	fixtures := regressionFixtures(t, db, w.ID)

	// Soft-delete one trigger: it must never appear in filtered results.
	require.NoError(t, manager.DeleteTrigger(t.Context(), fixtures["plain-hook"].ID))

	type testCase struct {
		name     string
		filter   string
		expected []string
	}

	testCases := []testCase{
		{
			name:     "empty filter returns all non-deleted triggers",
			filter:   "",
			expected: []string{"100%_coverage", "Payments-Refund", "hook -- drop comment", "o'brien-hook", "payments-sync", "x'; DELETE FROM triggers; --"},
		},
		{
			name:     "substring search",
			filter:   "ments-s",
			expected: []string{"payments-sync"},
		},
		{
			name:     "case-insensitive search matches both cases",
			filter:   "payment",
			expected: []string{"Payments-Refund", "payments-sync"},
		},
		{
			name:     "uppercase filter is case-insensitive",
			filter:   "PAYMENTS",
			expected: []string{"Payments-Refund", "payments-sync"},
		},
		{
			name:     "percent keeps wildcard semantics",
			filter:   "100%",
			expected: []string{"100%_coverage"},
		},
		{
			name:     "underscore keeps wildcard semantics",
			filter:   "100_",
			expected: []string{"100%_coverage"},
		},
		{
			name:     "percent and underscore combined",
			filter:   "100%_coverage",
			expected: []string{"100%_coverage"},
		},
		{
			name:     "apostrophe is inert data",
			filter:   "o'brien",
			expected: []string{"o'brien-hook"},
		},
		{
			name:     "comment-like text is inert data",
			filter:   "-- drop",
			expected: []string{"hook -- drop comment"},
		},
		{
			name:     "SQL-looking name is inert data",
			filter:   "DELETE FROM triggers",
			expected: []string{"x'; DELETE FROM triggers; --"},
		},
		{
			name:     "no match returns empty page",
			filter:   "zzz-no-such-trigger",
			expected: []string{},
		},
		{
			name:     "deleted trigger excluded from empty-filter listing",
			filter:   "plain",
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cursor, err := manager.ListTriggers(t.Context(), ListTriggersQuery{
				PageSize: 100,
				Options:  ListTriggerParams{Name: tc.filter},
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, triggerNames(cursor))
		})
	}
}

func TestListTriggersNameFilterCursorRegression(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	w := insertNoOpWorkflow(t, db)
	manager := NewManager(db, NewExpressionEvaluator(nil))

	// 5 triggers sharing the "cursor" prefix, plus one deleted one and one
	// out-of-filter trigger.
	for _, name := range []string{"cursor-aaa", "cursor-bbb", "cursor-ccc", "cursor-ddd", "cursor-eee", "other"} {
		insertNamedTrigger(t, db, w.ID, name)
	}
	deleted := insertNamedTrigger(t, db, w.ID, "cursor-deleted")
	require.NoError(t, manager.DeleteTrigger(t.Context(), deleted.ID))

	pageSize := uint64(2)
	query := ListTriggersQuery{
		PageSize: pageSize,
		Options:  ListTriggerParams{Name: "cursor"},
	}

	cursor, err := manager.ListTriggers(t.Context(), query)
	require.NoError(t, err)
	require.Len(t, cursor.Data, 2)
	require.True(t, cursor.HasMore)
	require.NotEmpty(t, cursor.Next)

	// The filter must be carried inside the cursor: decoding it exposes the
	// same name filter.
	var decoded ListTriggersQuery
	require.NoError(t, bunpaginate.UnmarshalCursor(cursor.Next, &decoded))
	require.Equal(t, "cursor", decoded.Options.Name)
	require.Equal(t, pageSize, decoded.PageSize)

	collected := triggerNames(cursor)

	// Walk the remaining pages through the cursor only.
	for cursor.HasMore {
		cursor, err = manager.ListTriggers(t.Context(), decoded)
		require.NoError(t, err)
		collected = append(collected, triggerNames(cursor)...)
		if !cursor.HasMore {
			break
		}
		require.NoError(t, bunpaginate.UnmarshalCursor(cursor.Next, &decoded))
	}

	slices.Sort(collected)
	require.Equal(t, []string{"cursor-aaa", "cursor-bbb", "cursor-ccc", "cursor-ddd", "cursor-eee"}, collected)
}
