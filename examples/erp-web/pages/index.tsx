import { Button, Card } from 'antd';
import { GGID_URL, CLIENT_ID, REDIRECT_URI } from '../lib/auth';

export default function Home() {
  const login = () => {
    const url = `${GGID_URL}/oauth/authorize?response_type=code&client_id=${CLIENT_ID}&redirect_uri=${encodeURIComponent(REDIRECT_URI)}&scope=openid profile email`;
    window.location.href = url;
  };

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
      <Card title="GGID ERP Demo" style={{ width: 400, textAlign: 'center' }}>
        <p>Sign in with your GGID account.</p>
        <Button type="primary" size="large" onClick={login} block>Login with GGID</Button>
      </Card>
    </div>
  );
}
