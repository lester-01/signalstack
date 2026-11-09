import { Avatar } from "./avatar";
import { shortId } from "@/lib/identity";
import { cn } from "@/lib/utils";

export function PresenceList({
  clients,
  selfId,
  recipient,
  onSelect,
}: {
  clients: string[];
  selfId: string;
  recipient: string;
  onSelect: (id: string) => void;
}) {
  const others = clients.filter((id) => id !== selfId);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <h2 className="px-5 pb-2 text-sm font-medium">
        Online <span className="font-mono text-muted">{clients.length}</span>
      </h2>
      <ul className="min-h-0 flex-1 space-y-0.5 overflow-y-auto px-2 pb-3">
        <li>
          <button
            type="button"
            onClick={() => onSelect("all")}
            aria-pressed={recipient === "all"}
            className={cn(
              "flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-sm transition-colors",
              "hover:bg-hover focus-visible:outline-2 focus-visible:outline-signal",
              recipient === "all" && "bg-hover font-medium"
            )}
          >
            <span
              aria-hidden="true"
              className="grid size-8 shrink-0 place-items-center rounded-full bg-ink text-xs font-semibold text-paper"
            >
              #
            </span>
            Everyone
          </button>
        </li>
        {others.map((id) => (
          <li key={id}>
            <button
              type="button"
              onClick={() => onSelect(id)}
              aria-pressed={recipient === id}
              title={`Message ${id} privately`}
              className={cn(
                "flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors",
                "hover:bg-hover focus-visible:outline-2 focus-visible:outline-signal",
                recipient === id && "bg-hover"
              )}
            >
              <Avatar id={id} />
              <span className="truncate font-mono text-[13px]">{shortId(id)}</span>
            </button>
          </li>
        ))}
        {others.length === 0 && (
          <li className="px-3 py-2 text-sm text-muted">
            Nobody else is here. Open a second tab to chat with yourself.
          </li>
        )}
      </ul>
    </div>
  );
}
