package dto

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

func NewErrorResponse(err string, message string, code int) *ErrorResponse {
	return &ErrorResponse{
		Error:   err,
		Message: message,
		Code:    code,
	}
}
