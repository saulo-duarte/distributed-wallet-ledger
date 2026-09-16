package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const MaxRequestBodySize = 1 << 20

var (
	ErrInvalidRequestBody = errors.New("invalid request body")
	ErrMultipleJSONValues = errors.New("request body must contain only one JSON object")
)

type ErrorResponse struct {
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func DecodeJSON(
	w http.ResponseWriter,
	r *http.Request,
	destination any,
) error {
	decoder := json.NewDecoder(
		http.MaxBytesReader(w, r.Body, MaxRequestBodySize),
	)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return ErrInvalidRequestBody
	}

	var extra any
	err := decoder.Decode(&extra)
	if err == nil {
		return ErrMultipleJSONValues
	}
	if !errors.Is(err, io.EOF) {
		return ErrInvalidRequestBody
	}

	return nil
}

func WriteJSON(
	w http.ResponseWriter,
	statusCode int,
	payload any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(
	w http.ResponseWriter,
	statusCode int,
	code string,
	message string,
	requestID string,
) {
	WriteJSON(
		w,
		statusCode,
		ErrorResponse{
			Error:     message,
			Code:      code,
			RequestID: requestID,
		},
	)
}
