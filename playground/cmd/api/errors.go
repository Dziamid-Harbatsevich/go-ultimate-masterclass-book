package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSONErrorResponse creates a uniform error contract matching standard microservice criteria
type JSONErrorResponse struct {
	Status int               `json:"status"`
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

// writeErrorPayload streams formatted error data down the active network pipe
func writeErrorPayload(w http.ResponseWriter, status int, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	payload := JSONErrorResponse{
		Status: status,
		Error:  message,
		Fields: fields,
	}

	json.NewEncoder(w).Encode(payload)
}

func errorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeErrorPayload(w, status, message, nil)
}

func serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("❌ SERVER DESCRIPTOR FAILURE: %s %s | error: %v", r.Method, r.URL.Path, err)
	message := "The server encountered an internal processing breakdown"
	errorResponse(w, r, http.StatusInternalServerError, message)
}

func badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	writeErrorPayload(w, http.StatusUnprocessableEntity, "schema validation rules failed", errors)
}
