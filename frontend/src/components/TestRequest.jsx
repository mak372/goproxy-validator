import { useState } from "react";

const SERVICE_A = import.meta.env.VITE_SERVICE_A_URL || "http://localhost:8001";

export default function TestRequest() {
  const [method, setMethod] = useState("POST");
  const [endpoint, setEndpoint] = useState("");
  const [body, setBody] = useState("");
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setResult(null);

    if (method !== "GET" && method !== "DELETE") {
      try {
        JSON.parse(body);
      } catch {
        setResult({ ok: false, violations: null, message: "Request body is not valid JSON" });
        return;
      }
    }

    setLoading(true);
    try {
      const options = {
        method,
        headers: { "Content-Type": "application/json" },
      };
      if (method !== "GET" && method !== "DELETE") {
        options.body = body;
      }

      const res = await fetch(`${SERVICE_A}${endpoint}`, options);
      const data = await res.json();

      if (res.ok) {
        setResult({ ok: true, message: "Request passed validation", data });
      } else if (res.status === 400) {
        setResult({ ok: false, message: "Request violates contract", violations: data.violations });
      } else if (res.status === 502) {
        setResult({ ok: false, message: "Response from upstream violates contract", violations: data.violations });
      } else if (res.status === 404) {
        setResult({ ok: false, message: data.error || "No contract found for this endpoint", violations: null });
      } else {
        setResult({ ok: false, message: JSON.stringify(data), violations: null });
      }
    } catch (err) {
      setResult({ ok: false, message: "Could not reach proxy: " + err.message, violations: null });
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="panel">
      <h2>Test Request</h2>
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
          <label>Request Body (JSON)</label>
          <textarea
            rows={8}
            placeholder={'{\n  "name": "john",\n  "age": 25\n}'}
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
        </div>

        <button className="btn btn-primary" type="submit" disabled={loading}>
          {loading ? "Sending..." : "Send Request"}
        </button>
      </form>

      {result && (
        <div className={`result ${result.ok ? "success" : "error"}`}>
          <p className="result-title">{result.message}</p>
          {result.violations && result.violations.length > 0 && (
            <table className="violations-table">
              <thead>
                <tr>
                  <th>Field</th>
                  <th>Issue</th>
                  <th>Expected</th>
                  <th>Got</th>
                </tr>
              </thead>
              <tbody>
                {result.violations.map((v, i) => (
                  <tr key={i}>
                    <td>{v.Field}</td>
                    <td>{v.Issue}</td>
                    <td>{v.Expected}</td>
                    <td>{v.Got}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          {result.ok && result.data && (
            <pre>{JSON.stringify(result.data, null, 2)}</pre>
          )}
        </div>
      )}
    </div>
  );
}
