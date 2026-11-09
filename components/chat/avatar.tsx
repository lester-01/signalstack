import { idColor, idInitials } from "@/lib/identity";
import { cn } from "@/lib/utils";

export function Avatar({
  id,
  className,
}: {
  id: string;
  className?: string;
}) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "grid size-8 shrink-0 place-items-center rounded-full font-mono text-[11px] font-semibold text-white",
        className
      )}
      style={{ backgroundColor: idColor(id) }}
    >
      {idInitials(id)}
    </span>
  );
}
