# Contract-Validating Proxy

A Go project that demonstrates a **contract-validating reverse proxy** sitting between two services. The proxy intercepts all traffic, validates the request and response payloads against a defined contract schema, and forwards everything transparently.

Contracts are loaded **dynamically at runtime** — no file edits or restarts needed.

---

## How It Works

```
You (curl)
    ↓
POST /contract         POST /contract
    ↓                       ↓
Service A (:8001)  →  Proxy (:8080)  →  Service B (:8002)
                       ↕ validate              ↕
                      (in memory)          responds
```

1. You POST a contract to both **Service A** and the **Proxy** — they store it in memory
2. You POST dynamic data to **Service A** `/send`
3. **Service A** checks all contract fields are present in your payload, then forwards to the proxy
4. **Proxy** validates the request payload against its contract (field names + types)
5. **Proxy** forwards to **Service B**
6. **Service B** processes and responds
7. **Proxy** captures the response, validates it against the contract, and passes it back
8. **Service A** returns the final status to you

Nothing is blocked or modified — the proxy observes and reports violations only.

---

## Project Structure

```
go_project/
├── serviceA/
│   └── main.go          # Accepts dynamic contract + data, forwards to proxy
├── serviceB/
│   └── main.go          # Receives and processes the request
├── proxy/
│   └── main.go          # Intercepts, validates, and forwards traffic
├── validator/
│   └── validator.go     # Validates JSON payloads against a schema
├── config/
│   └── config.go        # Contract type definition and file loader
└── go.mod
```

---

## Contract Schema

A contract defines the endpoint, HTTP method, and expected fields + types for both request and response.

```json
{
  "endpoint": "/api/user",
  "method": "POST",
  "request": {
    "user_id": "string",
    "email": "string",
    "age": "number"
  },
  "response": {
    "status": "string",
    "message": "string"
  }
}
```

Supported types: `string`, `number`, `boolean`, `object`, `array`, `null`

---

## Services

| Service   | Port | Role                                            |
|-----------|------|-------------------------------------------------|
| Service A | 8001 | Accepts contract + dynamic data, sends to proxy |
| Proxy     | 8080 | Validates and forwards traffic                  |
| Service B | 8002 | Receives request, returns JSON response         |

### Service A Endpoints

| Method | Path        | Description                                      |
|--------|-------------|--------------------------------------------------|
| POST   | `/contract` | Load a contract into Service A                   |
| GET    | `/schema`   | View the currently loaded contract               |
| POST   | `/send`     | Send dynamic data to the proxy via the contract  |

### Proxy Endpoints

| Method | Path        | Description                                      |
|--------|-------------|--------------------------------------------------|
| POST   | `/contract` | Load a contract into the proxy                   |
| ANY    | `/*`        | Intercept, validate, and forward to Service B    |

---

## Running the Project

Open three terminals and run each service:

```bash
# Terminal 1 — Service B
cd serviceB && go run main.go

# Terminal 2 — Proxy
cd proxy && go run main.go

# Terminal 3 — Service A
cd serviceA && go run main.go
```

---

## Workflow

### Step 1 — POST the contract to both services

```bash
curl -X POST http://localhost:8001/contract \
  -H "Content-Type: application/json" \
  -d '{"endpoint":"/api/user","method":"POST","request":{"user_id":"string","email":"string","age":"number"},"response":{"status":"string","message":"string"}}'

curl -X POST http://localhost:8080/contract \
  -H "Content-Type: application/json" \
  -d '{"endpoint":"/api/user","method":"POST","request":{"user_id":"string","email":"string","age":"number"},"response":{"status":"string","message":"string"}}'
```

### Step 2 — (Optional) Check the loaded schema

```bash
curl http://localhost:8001/schema
```

### Step 3 — Send dynamic data

```bash
curl -X POST http://localhost:8001/send \
  -H "Content-Type: application/json" \
  -d '{"user_id":"456","email":"alice@example.com","age":30}'
```

To change the contract later, repeat Step 1 with the new contract — no restart needed.

---

## Example Output

### Proxy — valid request

```
Contract updated: POST /api/user
Proxy running on :8080

=== INCOMING REQUEST ===
Endpoint: POST /api/user
Body: {"user_id":"456","email":"alice@example.com","age":30}
[REQUEST] Contract OK

=== OUTGOING RESPONSE ===
Status: 200
Body: {"status":"success","message":"user processed"}
[RESPONSE] Contract OK
========================
```

### Proxy — violations

```
[REQUEST] Contract VIOLATIONS FOUND:
  - Field: age   | Issue: wrong type    | Expected: number | Got: string
  - Field: email | Issue: missing field | Expected: string | Got: null
```

### Service A — missing field rejection

```
HTTP 400: missing contract fields: email
```

---

## Requirements

- Go 1.24+

---

## Tech Stack

- **Language:** Go
- **Proxy:** `net/http/httputil.ReverseProxy`
- **Validation:** Custom JSON schema validator
- **Contract loading:** Dynamic via HTTP (in-memory, no file required)
