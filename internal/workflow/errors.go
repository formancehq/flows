package workflow

import (
	"errors"

	"go.temporal.io/sdk/temporal"
)

// WorkflowNotFoundErrorType crosses the Temporal activity/child boundaries.
const WorkflowNotFoundErrorType = "WORKFLOW_NOT_FOUND"

func IsWorkflowNotFoundError(err error) bool {
	if errors.Is(err, ErrWorkflowNotFound) {
		return true
	}
	appErr, ok := errors.AsType[*temporal.ApplicationError](err)
	return ok && appErr.Type() == WorkflowNotFoundErrorType
}
