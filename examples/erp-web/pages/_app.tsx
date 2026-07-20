import { ConfigProvider } from 'antd';
import type { AppProps } from 'next/app';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <ConfigProvider theme={{ token: { colorPrimary: '#1677ff' } }}>
      <Component {...pageProps} />
    </ConfigProvider>
  );
}
