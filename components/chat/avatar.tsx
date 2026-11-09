import { idColor, idInitials } from "@/lib/identity";
import { cn } from "@/lib/utils";

export function Avatar({ id, className }: { id: string; className?: string }) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "grid size-9 shrink-0 place-items-center rounded-full font-mono text-[11px] font-semibold text-white",
        className
      )}
      style={{ backgroundColor: idColor(id) }}
    >
      {idInitials(id)}
    </span>
  );
}

/** Overlapping avatars for the people online, with a +N for the rest. */
export function AvatarStack({ ids, max = 4 }: { ids: string[]; max?: number }) {
  const shown = ids.slice(0, max);
  const extra = ids.length - shown.length;
  return (
    <span className="flex items-center">
      {shown.map((id) => (
        <Avatar
          key={id}
          id={id}
          className="-ml-2 size-7 text-[9px] ring-2 ring-panel first:ml-0"
        />
      ))}
      {extra > 0 && (
        <span className="-ml-2 grid size-7 place-items-center rounded-full bg-hover font-mono text-[10px] text-muted ring-2 ring-panel">
          +{extra}
        </span>
      )}
    </span>
  );
}
