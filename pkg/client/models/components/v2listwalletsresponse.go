// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type V2ListWalletsResponseCursor struct {
	PageSize int64      `json:"pageSize"`
	HasMore  *bool      `json:"hasMore,omitempty"`
	Previous *string    `json:"previous,omitempty"`
	Next     *string    `json:"next,omitempty"`
	Data     []V2Wallet `json:"data"`
}

func (o *V2ListWalletsResponseCursor) GetPageSize() int64 {
	if o == nil {
		return 0
	}
	return o.PageSize
}

func (o *V2ListWalletsResponseCursor) GetHasMore() *bool {
	if o == nil {
		return nil
	}
	return o.HasMore
}

func (o *V2ListWalletsResponseCursor) GetPrevious() *string {
	if o == nil {
		return nil
	}
	return o.Previous
}

func (o *V2ListWalletsResponseCursor) GetNext() *string {
	if o == nil {
		return nil
	}
	return o.Next
}

func (o *V2ListWalletsResponseCursor) GetData() []V2Wallet {
	if o == nil {
		return []V2Wallet{}
	}
	return o.Data
}

type V2ListWalletsResponse struct {
	Cursor V2ListWalletsResponseCursor `json:"cursor"`
}

func (o *V2ListWalletsResponse) GetCursor() V2ListWalletsResponseCursor {
	if o == nil {
		return V2ListWalletsResponseCursor{}
	}
	return o.Cursor
}
