package rbody

import (
	"encoding/json"
	"fmt"
)

type ApiResponse struct {
	Meta *ResponseMeta   `json:"meta"`
	Body json.RawMessage `json:"body"`
}

type ResponseMeta struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (r *ApiResponse) Error() error {
	if r.Meta.Code == 200 {
		return nil
	}
	return fmt.Errorf("%d: %s", r.Meta.Code, r.Meta.Message)
}

func OkResp(t string, body interface{}) *ApiResponse {
	bRaw, err := json.Marshal(body)
	if err != nil {
		return ErrResp(500, err)
	}
	resp := &ApiResponse{
		Meta: &ResponseMeta{
			Code:    200,
			Message: "success",
			Type:    t,
		},
		Body: json.RawMessage(bRaw),
	}
	return resp
}

func ErrResp(code int, err error) *ApiResponse {
	resp := &ApiResponse{
		Meta: &ResponseMeta{
			Code:    code,
			Message: err.Error(),
			Type:    "error",
		},
		Body: nil,
	}
	return resp
}
