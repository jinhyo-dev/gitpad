/**
 * The gitpad mark: two commits on a lane and a curve into a third node — the
 * same shape the TUI draws with ●╮ / ●╰● in its header.
 */
export function Logo({ size = 22, className = "" }: { size?: number; className?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 64 64" aria-hidden="true" className={className}>
      <rect width="64" height="64" rx="14" fill="#1e1e2e" />
      <path
        d="M22 24v16M22 24c0 8 20 0 20 8"
        stroke="#89b4fa"
        strokeWidth="4"
        fill="none"
        strokeLinecap="round"
      />
      <circle cx="22" cy="18" r="6" fill="#89b4fa" />
      <circle cx="22" cy="46" r="6" fill="#89b4fa" />
      <circle cx="42" cy="32" r="6" fill="#cba6f7" />
    </svg>
  );
}
