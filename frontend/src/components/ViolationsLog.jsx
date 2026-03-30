import { useState, useEffect } from "react";

const PROXY = import.meta.env.VITE_PROXY_URL || "http://localhost:8080";

export default function ViolationsLog() {
  const [violations, setViolations] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch(`${PROXY}/violations`)
      .then((res) => res.json())
      .then((data) => {
        setViolations(data);
        setLoading(false);
      })
      .catch((err) => {
        setError("Could not reach proxy: " + err.message);
        setLoading(false);
      });
  }, []);

  if (loading) return <div className="panel"><p>Loading violations...</p></div>;
  if (error) return <div className="panel"><p className="error-text">{error}</p></div>;

  return (
    <div className="panel violations-panel">
      <h2>Violation History</h2>
      {violations.length === 0 ? (
        <p className="empty-text">No violations recorded.</p>
      ) : (
        <div className="violation-list">
          {[...violations].reverse().map((record, i) => (
            <div key={i} className="violation-record">
              <div className="violation-record-header">
                <span className={`badge ${record.direction === "REQUEST" ? "badge-request" : "badge-response"}`}>
                  {record.direction}
                </span>
                <span className="violation-endpoint">{record.method} {record.endpoint}</span>
                <span className="violation-time">{new Date(record.timestamp).toLocaleString()}</span>
              </div>
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
                  {record.violations.map((v, j) => (
                    <tr key={j}>
                      <td>{v.Field}</td>
                      <td>{v.Issue}</td>
                      <td>{v.Expected}</td>
                      <td>{v.Got}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
