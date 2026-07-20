import '../styles/globals.css';
import type { AppProps } from 'next/app';
import { ConfigProvider } from 'antd';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <ConfigProvider theme={{ token: { colorPrimary: '#4f46e5' } }}>
      <Component {...pageProps} />
    </ConfigProvider>
  );
}
