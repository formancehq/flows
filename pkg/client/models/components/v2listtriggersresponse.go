// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type V2ListTriggersResponseCursor struct {
	PageSize int64       `json:"pageSize"`
	HasMore  bool        `json:"hasMore"`
	Previous *string     `json:"previous,omitempty"`
	Next     *string     `json:"next,omitempty"`
	Data     []V2Trigger `json:"data"`
}

func (o *V2ListTriggersResponseCursor) GetPageSize() int64 {
	if o == nil {
		return 0
	}
	return o.PageSize
}

func (o *V2ListTriggersResponseCursor) GetHasMore() bool {
	if o == nil {
		return false
	}
	return o.HasMore
}

func (o *V2ListTriggersResponseCursor) GetPrevious() *string {
	if o == nil {
		return nil
	}
	return o.Previous
}

func (o *V2ListTriggersResponseCursor) GetNext() *string {
	if o == nil {
		return nil
	}
	return o.Next
}

func (o *V2ListTriggersResponseCursor) GetData() []V2Trigger {
	if o == nil {
		return []V2Trigger{}
	}
	return o.Data
}

type V2ListTriggersResponse struct {
	Cursor V2ListTriggersResponseCursor `json:"cursor"`
}

func (o *V2ListTriggersResponse) GetCursor() V2ListTriggersResponseCursor {
	if o == nil {
		return V2ListTriggersResponseCursor{}
	}
	return o.Cursor
}
