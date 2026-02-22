package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	Pagination *Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

func JSON(w http.ResponseWriter, statusCode int, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Success: success,
		Message: message,
		Data:    data,
	})
}

func Success(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusOK, true, message, data)
}

func Created(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusCreated, true, message, data)
}

func BadRequest(w http.ResponseWriter, message string, errors interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Message: message,
		Errors:  errors,
	})
}

func Unauthorized(w http.ResponseWriter, message string) {
	JSON(w, http.StatusUnauthorized, false, message, nil)
}

func Forbidden(w http.ResponseWriter, message string) {
	JSON(w, http.StatusForbidden, false, message, nil)
}

func NotFound(w http.ResponseWriter, message string) {
	JSON(w, http.StatusNotFound, false, message, nil)
}

func InternalError(w http.ResponseWriter, message string) {
	JSON(w, http.StatusInternalServerError, false, message, nil)
}

func Paginated(w http.ResponseWriter, message string, data interface{}, pagination *Pagination) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PaginatedResponse{
		Success:    true,
		Message:    message,
		Data:       data,
		Pagination: pagination,
	})
}