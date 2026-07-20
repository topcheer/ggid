import { ReactNode } from 'react';
import '../../styles/globals.css';

export const metadata = {
  title: '跨境ERP系统',
  description: '跨境电商ERP - GGID IAM 集成',
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>{children}</body>
    </html>
  );
}