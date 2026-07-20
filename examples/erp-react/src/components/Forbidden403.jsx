export default function Forbidden403({ perm }) {
  return (
    <div style={{ textAlign: 'center', padding: 40, fontFamily: 'system-ui' }}>
      <h1 style={{ fontSize: 48, color: '#ff4d4f' }}>403</h1>
      <p style={{ color: '#666' }}>You need permission: <code style={{ background: '#f5f5f5', padding: '2px 6px', borderRadius: 3 }}>{perm}</code></p>
      <a href="/dashboard" style={{ color: '#1890ff', textDecoration: 'none' }}>Back to Dashboard</a>
    </div>
  );
}