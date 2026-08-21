package clinicotp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type CodeService interface {
	SendCode(context.Context, string, string) error
	VerifyCode(context.Context, string, string, string) error
}

type LoginHandler struct {
	codes CodeService
}

func NewLoginHandler(codes CodeService) *LoginHandler {
	return &LoginHandler{codes: codes}
}

type loginRequest struct {
	Phone          string `json:"phone"`
	Code           string `json:"code,omitempty"`
	AppointmentRef string `json:"appointment_ref"`
}

type loginResult struct {
	State        string `json:"state"`
	Next         string `json:"next"`
	Notification string `json:"notification"`
}

func (h *LoginHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/code/request", h.requestCode)
	mux.HandleFunc("POST /login/code/verify", h.verifyCode)
	return mux
}

func (h *LoginHandler) requestCode(w http.ResponseWriter, r *http.Request) {
	requestID := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	var input loginRequest
	if requestID == "" || decodeJSON(w, r, &input) != nil || !validInput(input, false) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone, appointment_ref, and Idempotency-Key are required"})
		return
	}
	if err := h.codes.SendCode(r.Context(), input.Phone, requestID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, loginResult{
		State:        "code_sent",
		Next:         "verify_code",
		Notification: "Your secure sign-in code was sent. Do not share it.",
	})
}

func (h *LoginHandler) verifyCode(w http.ResponseWriter, r *http.Request) {
	requestID := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	var input loginRequest
	if requestID == "" || decodeJSON(w, r, &input) != nil || !validInput(input, true) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "phone, code, appointment_ref, and Idempotency-Key are required"})
		return
	}
	if err := h.codes.VerifyCode(r.Context(), input.Phone, input.Code, requestID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResult{
		State:        "verified",
		Next:         "open_appointment",
		Notification: "Identity confirmed. Continue to your appointment workspace.",
	})
}

func validInput(input loginRequest, needsCode bool) bool {
	if strings.TrimSpace(input.Phone) == "" || strings.TrimSpace(input.AppointmentRef) == "" {
		return false
	}
	return !needsCode || strings.TrimSpace(input.Code) != ""
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeServiceError(w http.ResponseWriter, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		writeJSON(w, apiErr.HTTPStatus, map[string]string{"error": apiErr.Code})
		return
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "message service unavailable"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
