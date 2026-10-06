// Manually added to match the additive OpenAPI pause projection; regenerate with Speakeasy when available.

package components

import "openapi/internal/utils"

type V2ActivityPause struct {
	ActivityID      string  `json:"activityID"`
	TemporalRunID   string  `json:"temporalRunID"`
	Name            string  `json:"name"`
	Attempt         int64   `json:"attempt"`
	LastFailure     *string `json:"lastFailure,omitempty"`
	LastFailureType *string `json:"lastFailureType,omitempty"`
	Reason          *string `json:"reason,omitempty"`
	MaxAttempts     *int64  `json:"maxAttempts,omitempty"`
}

func (a V2ActivityPause) MarshalJSON() ([]byte, error) { return utils.MarshalJSON(a, "", false) }

func (a *V2ActivityPause) UnmarshalJSON(data []byte) error {
	return utils.UnmarshalJSON(data, &a, "", false, false)
}

func (a *V2ActivityPause) GetActivityID() string {
	if a == nil {
		return ""
	}
	return a.ActivityID
}

func (a *V2ActivityPause) GetTemporalRunID() string {
	if a == nil {
		return ""
	}
	return a.TemporalRunID
}

func (a *V2ActivityPause) GetName() string {
	if a == nil {
		return ""
	}
	return a.Name
}

func (a *V2ActivityPause) GetAttempt() int64 {
	if a == nil {
		return 0
	}
	return a.Attempt
}

func (a *V2ActivityPause) GetLastFailure() *string {
	if a == nil {
		return nil
	}
	return a.LastFailure
}

func (a *V2ActivityPause) GetLastFailureType() *string {
	if a == nil {
		return nil
	}
	return a.LastFailureType
}

func (a *V2ActivityPause) GetReason() *string {
	if a == nil {
		return nil
	}
	return a.Reason
}

func (a *V2ActivityPause) GetMaxAttempts() *int64 {
	if a == nil {
		return nil
	}
	return a.MaxAttempts
}
