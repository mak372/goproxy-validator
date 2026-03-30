import { useState } from "react";

const PROXY = import.meta.env.VITE_PROXY_URL || "http://localhost:8080";

export default function ContractForm() {
  const [method, setMethod] = useState("POST");
  const [endpoint, setEndpoint] = useState("");
  const [target, setTarget] = useState("");
  const [requestSchema, setRequestSchema] = useState("");
  const [responseSchema, setResponseSchema] = useState("");
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setResult(null);

    let parsedRequest, parsedResponse;

    try {
      parsedRequest = JSON.parse(requestSchema);
    } catch {
      setResult({ ok: false, message: "Request schema is not valid JSON" });
      return;
    }

    try {
      parsedResponse = JSON.parse(responseSchema);
    } catch {
      setResult({ ok: false, message: "Response schema is not valid JSON" });
      return;
    }

    const contract = {
      method,
      endpoint,
      target,
      request: parsedRequest,
      response: parsedResponse,
    };

    setLoading(true);
    try {
      const res = await fetch(`${PROXY}/contract`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(contract),
      });
      const data = await res.text();
      if (res.ok) {
        setResult({ ok: true, message: `Contract published: ${method} ${endpoint}` });
      } else {
        setResult({ ok: false, message: data });
      }
    } catch (err) {
      setResult({ ok: false, message: "Could not reach proxy: " + err.message });
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="panel">
      <h2>Publish Contract</h2>
      <form onSubmit={handleSubmit}>
        <div className="row">
          <div className="field">
            <label>Method</label>
            <select value={method} onChange={(e) => setMethod(e.target.value)}>
              <option>GET</option>
              <option>POST</option>
              <option>PUT</option>
              <option>DELETE</option>
            </select>
          </div>
          <div className="field flex1">
            <label>Endpoint</label>
            <input
              type="text"
              placeholder="/api/user"
              value={endpoint}
              onChange={(e) => setEndpoint(e.target.value)}
              required
            />
          </div>
        </div>

        <div className="field">
          <label>Target (upstream URL)</label>
          <input
            type="text"
            placeholder="http://localhost:8002"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            required
          />
        </div>

        <div className="field">
          <label>Request Schema (JSON)</label>
          <textarea
            rows={6}
            placeholder={'{\n  "name": "string",\n  "age": "number"\n}'}
            value={requestSchema}
            onChange={(e) => setRequestSchema(e.target.value)}
            required
          />
        </div>

        <div className="field">
          <label>Response Schema (JSON)</label>
          <textarea
            rows={6}
            placeholder={'{\n  "status": "string",\n  "message": "string"\n}'}
            value={responseSchema}
            onChange={(e) => setResponseSchema(e.target.value)}
            required
          />
        </div>

        <button className="btn btn-primary" type="submit" disabled={loading}>
          {loading ? "Publishing..." : "Publish Contract"}
        </button>
      </form>

      {result && (
        <div className={`result ${result.ok ? "success" : "error"}`}>
          {result.message}
        </div>
      )}
    </div>
  );
}
