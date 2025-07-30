package client

import (
	"net/url"

	"github.com/notioncodes/types"
)

// DatabaseQueryRequestBody represents the body of a database query request.
type DatabaseQueryRequestBody struct {
	Filter      interface{} `json:"filter,omitempty"`
	Sorts       interface{} `json:"sorts,omitempty"`
	StartCursor *string     `json:"start_cursor,omitempty"`
	PageSize    *int        `json:"page_size,omitempty"`
}

// DatabaseQueryRequest represents a request to query a database.
type DatabaseQueryRequest struct {
	DatabaseID types.DatabaseID
	Body       DatabaseQueryRequestBody
}

func (dqr *DatabaseQueryRequest) GetPath() string {
	return "/databases/" + dqr.DatabaseID.String() + "/query"
}

func (dqr *DatabaseQueryRequest) GetMethod() string {
	return "POST"
}

func (dqr *DatabaseQueryRequest) GetBody() DatabaseQueryRequestBody {
	return dqr.Body
}

func (dqr *DatabaseQueryRequest) GetQuery() url.Values {
	return nil
}

func (dqr *DatabaseQueryRequest) Validate() error {
	return dqr.DatabaseID.Validate()
}

func (dqr *DatabaseQueryRequest) SetStartCursor(cursor *string) {
	dqr.Body.StartCursor = cursor
}

func (dqr *DatabaseQueryRequest) SetPageSize(pageSize *int) {
	dqr.Body.PageSize = pageSize
}
