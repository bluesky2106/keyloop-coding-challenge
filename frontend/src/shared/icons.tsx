// Decorative icons. They are always hidden from assistive technology: the text beside them
// carries the meaning, so a signal never depends on the icon or its colour alone.
interface IconProps {
  className?: string
}

export function WarningIcon({ className }: IconProps) {
  return (
    <svg
      className={className}
      viewBox="0 0 16 16"
      width="1em"
      height="1em"
      fill="currentColor"
      aria-hidden="true"
      focusable="false"
    >
      <path d="M8 1.2a1 1 0 0 1 .87.5l6.2 10.8a1 1 0 0 1-.87 1.5H1.8a1 1 0 0 1-.87-1.5L7.13 1.7A1 1 0 0 1 8 1.2Zm-.75 4.3v3.8h1.5V5.5h-1.5Zm.75 5.1a.9.9 0 1 0 0 1.8.9.9 0 0 0 0-1.8Z" />
    </svg>
  )
}

export function BrandMark({ className }: IconProps) {
  return (
    <svg
      className={className}
      viewBox="0 0 32 32"
      width="32"
      height="32"
      aria-hidden="true"
      focusable="false"
    >
      <rect width="32" height="32" rx="8" fill="currentColor" />
      <path
        d="M7 20.5V17l2.1-5.2A2 2 0 0 1 11 10.5h10a2 2 0 0 1 1.9 1.3L25 17v3.5a1 1 0 0 1-1 1h-1.5a1 1 0 0 1-1-1V20h-11v.5a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1Zm4.2-7.7L9.6 16.5h12.8l-1.6-3.7a.5.5 0 0 0-.46-.3H11.7a.5.5 0 0 0-.5.3Z"
        fill="#fff"
      />
    </svg>
  )
}
