# Contract-Validating Proxy

A Go project that demonstrates a **contract-validating reverse proxy** sitting between two services. The proxy intercepts all traffic, validates the request and response payloads against a defined contract schema, and either forwards the request or blocks it with a structured violation report.

Contracts are loaded **dynamically at runtime** no file edits or restarts needed. Multiple contracts can be registered simultaneously, each routing to its own upstream target.

The proxy also acts as an **HTTP-to-gRPC gateway**: a contract can declare `"protocol": "grpc"` and embed a `.proto` schema, and the proxy will accept REST/JSON, translate it into a Protobuf message, call the downstream gRPC service, and translate the response back to JSON — with the same request/response contract validation and violation reporting as the HTTP path. See [gRPC Gateway Mode](#grpc-gateway-mode) below.

---

## Live Demo

You can find the live demo at https://microservice-contract-validator.vercel.app

The project is deployed and accessible at:

| Component | URL |
|-----------|-----|
| Frontend  | https://microservice-contract-validator.vercel.app |
| Backend (Proxy) | https://microservice-contract-validator.onrender.com |

The frontend is hosted on **Vercel** and the backend proxy is hosted on **Render** (free tier). Note that the backend may take ~30 seconds to respond on the first request after a period of inactivity (Render free tier spins down idle services).

## How It Works

```
You / Frontend
      ↓
Service A (:8001)  →  Proxy (:8080)  →  Service B (:8002)
                        ↕ validate
                     (in memory)
```

1. POST a contract to the **Proxy** it stores the contract and its upstream `target` in memory
2. Send a request to **Service A** `/verify`
3. **Service A** forwards to the proxy at the contract's endpoint
4. **Proxy** validates the request against the contract schema
   - If violations are found → returns `400` with a structured violation list (request is blocked)
5. **Proxy** forwards to **Service B**
6. **Service B** processes and responds
7. **Proxy** validates the response against the contract schema
   - If violations are found → returns `502` with a structured violation list
8. **Proxy** passes the response back to Service A, which returns it to the caller

All violations are logged to a file via Zap and stored in memory, accessible via `GET /violations`.

---

## Project Structure

```
go_project/
├── serviceA/
│   └── main.go          # KYC Verification Service — accepts requests, forwards to proxy
├── serviceB/
│   └── main.go          # Identity Registry Service (HTTP/JSON) — processes and responds
├── serviceB-grpc/
│   └── main.go          # Identity Registry Service (gRPC) — same logic, served over gRPC
├── proxy/
│   └── main.go          # Intercepts, validates, blocks or forwards traffic (HTTP or gRPC upstream)
├── grpcgateway/
│   └── grpcgateway.go   # Dynamic HTTP-to-gRPC translation (protocompile + dynamicpb, no codegen)
├── protos/kyc/kycpb/
│   └── kyc.proto        # Example .proto schema for the gRPC demo (+ generated Go stubs for serviceB-grpc)
├── validator/
│   └── validator.go     # Recursive JSON schema validator (supports nested objects + arrays)
├── config/
│   └── config.go        # Contract type definition and file loader
├── logger/
│   └── logger.go        # Zap-based structured logger
├── frontend/            # React frontend (Vite)
└── go.mod
```

---

## Contract Schema

A contract defines the endpoint, HTTP method, the upstream `target` URL, and expected fields + types for both request and response.

```json
{
  "endpoint": "/api/kyc/verify",
  "method": "POST",
  "target": "http://localhost:8002",
  "request": {
    "customerId": "string",
    "fullName": "string",
    "dateOfBirth": "string",
    "documentType": "string",
    "documentNumber": "string",
    "address": {
      "street": "string",
      "city": "string",
      "pincode": "string"
    }
  },
  "response": {
    "customerId": "string",
    "verificationId": "string",
    "status": "string",
    "riskScore": "number",
    "verifiedAt": "string"
  }
}
```

Supported types: `string`, `number`, `boolean`, `object`, `array`, `null`

Nested objects and arrays are validated recursively. Field paths in violation reports use dot notation (e.g. `address.city`) and bracket notation for array items (e.g. `items[0].name`).

---

## gRPC Gateway Mode

A contract can proxy to a **gRPC upstream** instead of an HTTP one. Add `"protocol": "grpc"` plus the contract's `.proto` schema:

```json
{
  "endpoint": "/api/kyc/verify-grpc",
  "method": "POST",
  "target": "localhost:9002",
  "protocol": "grpc",
  "grpcService": "kyc.KycService",
  "grpcMethod": "Verify",
  "protoSource": "syntax = \"proto3\"; package kyc; ...",
  "request": { "...": "same JSON schema as the HTTP contract" },
  "response": { "...": "same JSON schema as the HTTP contract" }
}
```

- `target` is a `host:port` gRPC address (no scheme).
- `protoSource` is the raw `.proto` file contents. It's compiled **in memory** at contract-registration time — no `protoc` binary, no generated stubs, no gateway rebuild or restart needed to add a new gRPC contract, matching the existing "dynamic contracts" model.
- `request`/`response` are validated exactly as they are for HTTP contracts; the transport underneath is transparent to the validation and violation-logging layer.

Request flow: **REST/JSON in → Protobuf message → gRPC call to the upstream → Protobuf response → JSON out**, with the same contract-violation blocking (`400`/`502` + structured violation list) on either side of the call.

Implementation notes (`grpcgateway/grpcgateway.go`):
- [`protocompile`](https://github.com/bufbuild/protocompile) (the compiler used inside the `buf` CLI) compiles the contract's `.proto` source into standard `google.golang.org/protobuf` descriptors — no third-party dynamic-message library, no deprecated APIs.
- [`dynamicpb`](https://pkg.go.dev/google.golang.org/protobuf/types/dynamicpb) builds request/response messages from those descriptors, and `protojson` converts between them and the JSON the gateway speaks to callers.
- `grpc.ClientConn.Invoke` dispatches the call generically by fully-qualified method name — no generated client stub is needed for the gateway to call an arbitrary gRPC service.
- Compiled descriptors and dialed connections are cached per contract key, so a contract's `.proto` is parsed once (at first use), not on every request.

The demo downstream (`serviceB-grpc/`), by contrast, is a normal `protoc`/`buf`-generated gRPC server — real backend teams write services against generated stubs for type safety and performance; only the gateway needs to handle schemas it doesn't know about at compile time.

### Try it

```bash
# 1. Start the gRPC downstream
cd serviceB-grpc && go run main.go   # :9002

# 2. Register a gRPC contract (protoSource = contents of protos/kyc/kycpb/kyc.proto)
curl -X POST http://localhost:8080/contract \
  -H "Content-Type: application/json" \
  -d @contracts/kyc-grpc-contract.json

# 3. Call it exactly like the HTTP contract
curl -X POST http://localhost:8080/api/kyc/verify-grpc \
  -H "Content-Type: application/json" \
  -d '{
    "customerId": "C001",
    "fullName": "Amit Sharma",
    "dateOfBirth": "1990-01-15",
    "documentType": "DL",
    "documentNumber": "DL1234567",
    "address": { "street": "12 MG Road", "city": "Bangalore", "pincode": "560001" }
  }'
```

To regenerate `serviceB-grpc`'s stubs after editing `protos/kyc/kycpb/kyc.proto`, run `buf generate` (requires the `buf`, `protoc-gen-go`, and `protoc-gen-go-grpc` binaries on `PATH` — all installable via `go install`, no `protoc` C++ binary required).

---

## Services

| Service   | Port | Role                                                      |
|-----------|------|-----------------------------------------------------------|
| Service A | 8001 | KYC Verification - accepts requests, forwards to proxy    |
| Proxy     | 8080 | Validates and forwards traffic; manages contracts         |
| Service B | 8002 | Identity Registry - processes KYC request, returns result |

### Service A Endpoints

| Method | Path      | Description                                      |
|--------|-----------|--------------------------------------------------|
| POST   | `/verify` | Accept a KYC request and forward it to the proxy |

### Proxy Endpoints

| Method | Path          | Description                                             |
|--------|---------------|---------------------------------------------------------|
| POST   | `/contract`   | Register a new contract (includes target URL)           |
| GET    | `/contracts`  | List all currently loaded contracts                     |
| GET    | `/violations` | Return the in-memory violation history                  |
| ANY    | `/*`          | Intercept, validate, and forward to the contract target |

---

## Running the Project

### Backend

Open three terminals and run each service:

```bash
# Terminal 1 — Service B (Identity Registry)
cd serviceB && go run main.go

# Terminal 2 — Proxy
cd proxy && go run main.go

# Terminal 3 — Service A (KYC Verification)
cd serviceA && go run main.go
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend runs on `http://localhost:5173` by default.

---

## Workflow (curl)

### Step 1 - Register a contract with the proxy

```bash
curl -X POST http://localhost:8080/contract \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint": "/api/kyc/verify",
    "method": "POST",
    "target": "http://localhost:8002",
    "request": {
      "customerId": "string",
      "fullName": "string",
      "dateOfBirth": "string",
      "documentType": "string",
      "documentNumber": "string",
      "address": {
        "street": "string",
        "city": "string",
        "pincode": "string"
      }
    },
    "response": {
      "customerId": "string",
      "verificationId": "string",
      "status": "string",
      "riskScore": "number",
      "verifiedAt": "string"
    }
  }'
```

### Step 2 - Send a KYC request

```bash
curl -X POST http://localhost:8001/verify \
  -H "Content-Type: application/json" \
  -d '{
    "customerId": "C001",
    "fullName": "Amit Sharma",
    "dateOfBirth": "1990-01-15",
    "documentType": "DL",
    "documentNumber": "DL1234567",
    "address": {
      "street": "12 MG Road",
      "city": "Bangalore",
      "pincode": "560001"
    }
  }'
```

### Step 3 - View violation history

```bash
curl http://localhost:8080/violations
```

---

## Example Output

### Proxy - valid request and response

```
=== INCOMING REQUEST ===
Endpoint: POST /api/kyc/verify
Body: {"customerId":"C001","fullName":"Amit Sharma",...}
[REQUEST] Contract OK

=== OUTGOING RESPONSE ===
Status: 200
Body: {"customerId":"C001","verificationId":"VER-...","status":"verified","riskScore":12.4,"verifiedAt":"..."}
[RESPONSE] Contract OK
========================
```

### Proxy - request blocked (violations found)

```
REQUEST blocked — contract violations found

HTTP 400:
{
  "error": "request violates contract",
  "violations": [
    { "Field": "address.pincode", "Issue": "missing field", "Expected": "string", "Got": "null" },
    { "Field": "fullName",        "Issue": "wrong type",    "Expected": "string", "Got": "number" }
  ]
}
```

### Proxy - response blocked (upstream violation)

```
HTTP 502:
{
  "error": "response from upstream violates contract",
  "violations": [
    { "Field": "verifiedAt", "Issue": "missing field", "Expected": "string", "Got": "null" }
  ]
}
```

---

## Frontend

The React frontend provides three panels:

- **Publish Contract** - define a contract (method, endpoint, target URL, request/response schemas) and register it with the proxy
- **Test Request** - send a request to Service A and view the result or a detailed violation table inline
- **Violation History** - browse all recorded violations with timestamps, direction badges (REQUEST / RESPONSE), and per-field details

Service URLs are configurable via environment variables:

```
VITE_PROXY_URL=http://localhost:8080
VITE_SERVICE_A_URL=http://localhost:8001
```

---



## Requirements

- Go 1.24+
- Node.js 18+ (frontend only)

---

## Tech Stack

- **Language:** Go
- **Proxy:** `net/http/httputil.ReverseProxy`
- **Validation:** Custom recursive JSON schema validator (no external dependencies)
- **Logging:** Uber Zap (structured, file-based)
- **Contract storage:** In-memory map, keyed by `METHOD /endpoint`; can also bootstrap from file (`contracts/user-service.json`)
- **Frontend:** React + Vite
- **Hosting:** Vercel (frontend) + Render (backend)
