import { type Viewport } from 'next';
import { Geist } from 'next/font/google';
import localFont from 'next/font/local';
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

// JetBrains Mono whole (SIL OFL 1.1, src/fonts/JetBrainsMono-OFL.txt), not
// Google's latin subset: that leaves out the box-drawing and block characters
// gnux draws its trees, frames and bars with, so a browser took them from
// another font, of another width and height, and the lines came out bent and
// broken. Every glyph here is 0.6em wide, and the lines reach 1.52em, so
// they join at any line height up to that.
const mono = localFont({
  src: [
    { path: '../fonts/JetBrainsMono-Regular.woff2', weight: '400', style: 'normal' },
    { path: '../fonts/JetBrainsMono-Medium.woff2', weight: '500', style: 'normal' },
    { path: '../fonts/JetBrainsMono-SemiBold.woff2', weight: '600', style: 'normal' },
    { path: '../fonts/JetBrainsMono-Bold.woff2', weight: '700', style: 'normal' }
  ],
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
