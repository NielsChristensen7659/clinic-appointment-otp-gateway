package clinicotp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeCodes struct {
	verifyErr error
	phone     string
	code      string
	key       string
}

func (f *fakeCodes) SendCode(_ context.Context, phone, key string) error {
	f.phone, f.key = phone, key
	return nil
}

func (f *fakeCodes) VerifyCode(_ context.Context, phone, code, key string) error {
	f.phone, f.code, f.key = phone, code, key
	return f.verifyErr
}

func TestAppointmentLoginDecision(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantState  string
		wantNext   string
	}{
		{name: "verified patient enters appointment", body: `{"phone":"+15551234567","code":"123456","appointment_ref":"appt_42"}`, wantStatus: http.StatusOK, wantState: "verified", wantNext: "open_appointment"},
		{name: "rejected code keeps appointment closed", body: `{"phone":"+15551234567","code":"000000","appointment_ref":"appt_42"}`, serviceErr: &APIError{Code: "CODE_REJECTED", HTTPStatus: http.StatusUnprocessableEntity}, wantStatus: http.StatusUnprocessableEntity},
		{name: "missing appointment context is rejected locally", body: `{"phone":"+15551234567","code":"123456"}`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codes := &fakeCodes{verifyErr: tt.serviceErr}
			req := httptest.NewRequest(http.MethodPost, "/login/code/verify", strings.NewReader(tt.body))
			req.Header.Set("Idempotency-Key", "login-attempt-42")
			res := httptest.NewRecorder()
			NewLoginHandler(codes).Routes().ServeHTTP(res, req)

			if res.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", res.Code, tt.wantStatus)
			}
			if tt.wantState != "" {
				var got loginResult
				if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if got.State != tt.wantState || got.Next != tt.wantNext {
					t.Fatalf("result = %#v", got)
				}
			}
		})
	}
}

func TestSMSClientDecodesEnvelopeBeforeStatus(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		response    string
		wantErr     bool
		wantAPI     bool
		wantRetries int
	}{
		{name: "success", status: http.StatusOK, response: `{"ok":true,"data":{"sent":true},"metadata":{}}`, wantRetries: 1},
		{name: "business rejection", status: http.StatusBadRequest, response: `{"ok":false,"data":null,"error":{"code":"INVALID_INPUT","message":"check input"},"metadata":{}}`, wantErr: true, wantAPI: true, wantRetries: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("Idempotency-Key") != "attempt-7" {
					t.Fatalf("unexpected request boundary")
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewSMSClient("test-key")
			client.baseURL = server.URL
			err := client.SendCode(context.Background(), "+15551234567", "attempt-7")
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v", err)
			}
			var apiErr *APIError
			if errors.As(err, &apiErr) != tt.wantAPI {
				t.Fatalf("APIError = %v", err)
			}
			if calls != tt.wantRetries {
				t.Fatalf("calls = %d, want %d", calls, tt.wantRetries)
			}
		})
	}
}
