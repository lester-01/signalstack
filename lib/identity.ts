/**
 * Every client ID maps to a fixed hue, so the same client has the same color
 * in the presence list, the avatars and the message log.
 */
export function idHue(id: string): number {
  let h = 2166136261; // FNV-1a
  for (let i = 0; i < id.length; i++) {
    h ^= id.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return (h >>> 0) % 360;
}

/** CSS color for an ID. --id-l is set per color scheme in globals.css. */
export function idColor(id: string): string {
  return `oklch(var(--id-l) 0.14 ${idHue(id)})`;
}

/** Two characters shown inside an avatar: the first hex digits after "c-". */
export function idInitials(id: string): string {
  return id.replace(/^c-/, "").slice(0, 2).toUpperCase();
}

/** Shortened ID for tight spaces ("c-382bcb"). The full ID stays in tooltips. */
export function shortId(id: string): string {
  return id.length > 10 ? id.slice(0, 8) : id;
}
