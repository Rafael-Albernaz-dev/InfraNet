export function ProductBrand() {
  return (
    <>
      <svg className="marca" aria-hidden="true" viewBox="0 0 36 36">
        <rect width="36" height="36" rx="5" fill="var(--brand-background)" />
        <g fill="var(--brand-foreground)" fillRule="evenodd">
          <path d="M5 29 L5 12 L11 5 L16 5 L29 24 L24 31 L20 25 L11 13 L11 29 Z" />
          <path d="M25 5 L31 5 L31 18 L25 12 Z" />
        </g>
      </svg>
      <strong>InfraNet</strong>
    </>
  );
}
