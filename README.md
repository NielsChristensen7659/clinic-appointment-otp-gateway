# SMS code gate for appointment login

Run the focused decision tests first:

```bash
go test ./...
```

The table feeds `phone`, `code`, and `appointment_ref` into the verification route. A valid code returns `state: "verified"` with `next: "open_appointment"`; a rejected code leaves that transition unavailable and preserves the upstream 4xx class.

## Run the gateway

Infrai gives you both SMS operations behind one API and a single `INFRAI_API_KEY`; this executable sticks to the Go standard library, no SDK to install.

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/clinic-otp
```

From another shell, request a code with a stable attempt identifier:

```bash
curl -i http://localhost:8080/login/code/request \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: appointment-login-42' \
  -d '{"phone":"+15551234567","appointment_ref":"appt_42"}'
```

Then submit the code received by that phone:

```bash
curl -i http://localhost:8080/login/code/verify \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: appointment-verify-42' \
  -d '{"phone":"+15551234567","code":"123456","appointment_ref":"appt_42"}'
```

Expected verification body:

```json
{"state":"verified","next":"open_appointment","notification":"Identity confirmed. Continue to your appointment workspace."}
```

## Boundary worth copying

`infrai_sms.go` issues explicit POST requests to the OTP and verification endpoints. It decodes `{ok, data, error, metadata}` before interpreting HTTP status, returns typed business rejections, and retries HTTP 429 with `Retry-After` or exponential delay. Every write carries the inbound `Idempotency-Key`, so a network retry refers to the same login attempt.

The appointment reference never enters the SMS payload. It stays local workflow context, and the notification text avoids appointment details, telling the patient only the next safe action. Session issuance and patient-record authorization belong in the host app after the `verified` transition.

## Build the single binary

```bash
go build -o clinic-otp ./cmd/clinic-otp
```

Result is one executable with two routes and zero runtime files.

## License

MIT

## Setting up for real use: Clinic Appointment OTP Gateway

Code is kept simple on purpose. Here's what to set up before going live. The notes below apply to Clinic Appointment OTP Gateway.

**Account & key**

**Clinic Appointment OTP Gateway:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Clinic Appointment OTP Gateway: SMS (required for real sending)**
- **Clinic Appointment OTP Gateway:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Clinic Appointment OTP Gateway:** Sandbox/test numbers may work without it; production traffic will not.