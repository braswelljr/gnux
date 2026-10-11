import { type Viewport } from 'next';
import { Geist, JetBrains_Mono } from 'next/font/google';
import { cn } from 'lib/utils';
import { siteConfig } from '@/config/site';
import Base from '@/providers/base';
import { createMetadata } from '@/utils/metadata';
import '@/styles/main.css';

export const metadata = createMetadata(siteConfig);

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#f9fafc' },
    { media: '(prefers-color-scheme: dark)', color: '#090a0f' }
  ]
};

const sans = Geist({
  subsets: ['latin'],
  display: 'swap',
  variable: '--font-geist'
});

const mono = JetBrains_Mono({
  subsets: ['latin'],
  display: 'swap',
  variable: '--font-geist-mono'
});

export default function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={cn(sans.variable, mono.variable)}
    >
      <body className="flex min-h-dvh flex-col font-sans">
        <Base>{children}</Base>
      </body>
    </html>
  );
}
