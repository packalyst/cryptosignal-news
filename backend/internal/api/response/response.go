package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse is the standard API response wrapper
type APIResponse struct {
	Data       interface{} `json:"data,omitempty"`
	Error      string      `json:"error,omitempty"`
	Query      string      `json:"query,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Meta       *Meta       `json:"meta,omitempty"`
}

// Pagination contains pagination information
type Pagination struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// Meta contains request metadata
type Meta struct {
	RequestID    string `json:"request_id"`
	ResponseTime int64  `json:"response_time_ms"`
}

// JSON writes a JSON response with the given status code
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// Log error but don't try to write again
			return
		}
	}
}

// Success writes a success response with data
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, APIResponse{
		Data: data,
	})
}

// SuccessWithPagination writes a success response with pagination
func SuccessWithPagination(w http.ResponseWriter, data interface{}, pagination *Pagination, meta *Meta) {
	JSON(w, http.StatusOK, APIResponse{
		Data:       data,
		Pagination: pagination,
		Meta:       meta,
	})
}

// SuccessWithQuery writes a success response with search query
func SuccessWithQuery(w http.ResponseWriter, data interface{}, query string, pagination *Pagination, meta *Meta) {
	JSON(w, http.StatusOK, APIResponse{
		Data:       data,
		Query:      query,
		Pagination: pagination,
		Meta:       meta,
	})
}

// Error writes an error response
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, APIResponse{
		Error: message,
	})
}

// NotFound writes a 404 not found response
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Resource not found"
	}
	Error(w, http.StatusNotFound, message)
}

// BadRequest writes a 400 bad request response
func BadRequest(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Bad request"
	}
	Error(w, http.StatusBadRequest, message)
}

// InternalError writes a 500 internal server error response
func InternalError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Internal server error"
	}
	Error(w, http.StatusInternalServerError, message)
}

// TooManyRequests writes a 429 rate limit exceeded response
func TooManyRequests(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Rate limit exceeded"
	}
	Error(w, http.StatusTooManyRequests, message)
}

// Created writes a 201 created response
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, APIResponse{
		Data: data,
	})
}

// NoContent writes a 204 no content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// NotModified writes a 304 not modified response
func NotModified(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotModified)
}

// NewPagination creates a new pagination struct
func NewPagination(total, limit, offset int) *Pagination {
	return &Pagination{
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}
}

// NewMeta creates a new meta struct
func NewMeta(requestID string, responseTimeMs int64) *Meta {
	return &Meta{
		RequestID:    requestID,
		ResponseTime: responseTimeMs,
	}
}
