import { logoSvg } from '@/components/ui/logo';

/** The mark in white on the card's dark ground, as a data URL Satori can draw. */
const LOGO_SRC = `data:image/svg+xml;base64,${Buffer.from(logoSvg('#f2f0eb')).toString('base64')}`;

/** The aurora's hues, written out: Satori cannot read the stylesheet's variables. */
const AURORA = { cyan: '#5cc8e6', violet: '#a07ef0', amber: '#f0a640' };

type OgCardProps = {
  title: string;
  description?: string;
  /** A small line above the title: the section, the site. */
  eyebrow: string;
  /** Shown bottom right: the host. */
  site: string;
};

/**
 * The Open Graph card, for next/og. It is rendered by Satori, not a browser:
 * inline styles only, flex layout, no CSS variables, so the aurora's colors
 * are written out.
 */
export function OgCard({ title, description, eyebrow, site }: OgCardProps) {
  return (
    <div
      style={{
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
        padding: 72,
        background: '#0e0f12',
        backgroundImage: `radial-gradient(circle at 85% 0%, ${AURORA.violet}55, transparent 55%), radial-gradient(circle at 10% 110%, ${AURORA.cyan}40, transparent 50%), radial-gradient(circle at 55% 40%, ${AURORA.amber}1f, transparent 45%)`,
        color: '#f4f5f8',
        fontFamily: 'sans-serif'
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 20 }}>
        <img
          alt=""
          width={72}
          height={72}
          src={LOGO_SRC}
          style={{ borderRadius: 16, boxShadow: '0 6px 0 0 #3a3b40' }}
        />
        <div style={{ display: 'flex', fontSize: 32, color: '#a4a8b8' }}>{eyebrow}</div>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
        <div
          style={{
            display: 'flex',
            fontSize: 76,
            fontWeight: 700,
            letterSpacing: -2,
            lineHeight: 1.05,
            color: '#ffffff'
          }}
        >
          {title}
        </div>
        {description && (
          <div style={{ display: 'flex', fontSize: 30, color: '#a4a8b8', lineHeight: 1.4 }}>
            {description}
          </div>
        )}
      </div>
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          fontSize: 24,
          color: '#7c8194'
        }}
      >
        <div style={{ display: 'flex' }}>$ gnux --list</div>
        <div style={{ display: 'flex' }}>{site}</div>
      </div>
    </div>
  );
}
