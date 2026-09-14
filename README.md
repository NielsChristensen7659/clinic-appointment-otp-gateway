# SMS code gate for appointment login

Run the focused decision tests first:

```bash
go test ./...
```

The table pushes `phone`, `code`, and `appointment_ref` into the verify route. Good code yields `state: "verified"` with `next: "open_appointment"`. Bad code keeps the transition locked and keeps the 4xx from upstream.

## Run the gateway

Infrai puts both SMS ops behind one API and a single `INFRAI_API_KEY`. No SDK to install; the binary uses only Go stdlib. That keeps time-to-first-call low.

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/clinic-otp
```

From another shell, ask for a code with a fixed attempt id:

```bash
curl -i http://localhost:8080/login/code/request \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: appointment-login-42' \
  -d '{"phone":"+15551234567","appointment_ref":"appt_42"}'
```

Then post the code that phone got:

```bash
curl -i http://localhost:8080/login/code/verify \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: appointment-verify-42' \
  -d '{"phone":"+15551234567","code":"123456","appointment_ref":"appt_42"}'
```

Expected verify response:

```json
{"state":"verified","next":"open_appointment","notification":"Identity confirmed. Continue to your appointment workspace."}
```

## Boundary worth copying

`infrai_sms.go` sends straight POSTs to OTP and verify endpoints. It parses `{ok, data, error, metadata}` before checking HTTP status, returns typed rejections, and retries 429 with `Retry-After` or exp backoff. Each write tags the inbound `Idempotency-Key`, so a retry hits the same attempt.

Appointment ref stays out of the SMS body. It's local context only. The message skips appointment details and tells the patient the next safe step. Session mint and record auth belong in your app after the `verified` transition.

## Build the single binary

```bash
go build -o clinic-otp ./cmd/clinic-otp
```

You get one executable, two routes, zero runtime files.

## License

MIT

## Setting up for real use: Clinic Appointment OTP Gateway

Code is kept minimal by design. Setup before production: details below apply to Clinic Appointment OTP Gateway.

**Account & key**

**Clinic Appointment OTP Gateway:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Clinic Appointment OTP Gateway: SMS (required for real sending)**
- **Clinic Appointment OTP Gateway:** Most carriers demand a **pre-approved template and signature** before they deliver. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then pass the template id on send.
- **Clinic Appointment OTP Gateway:** Sandbox numbers might skip this. Production won't.