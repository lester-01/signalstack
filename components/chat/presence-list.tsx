import { Avatar } from "./avatar";
import { shortId } from "@/lib/identity";
import { cn } from "@/lib/utils";

const row =
  "flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors hover:bg-hover focus-visible:outline-2 focus-visible:outline-signal";

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
    <ul className="space-y-1">
      <li>
        <button
          type="button"
          onClick={() => onSelect("all")}
          aria-pressed={recipient === "all"}
          className={cn(row, recipient === "all" && "bg-hover")}
        >
          <span
            aria-hidden="true"
            className="grid size-9 shrink-0 place-items-center rounded-full bg-ink font-semibold text-paper"
          >
            #
          </span>
          <span className="flex-1 text-[15px] font-medium">Everyone</span>
          <span className="font-mono text-xs text-muted">{clients.length}</span>
        </button>
      </li>
      {others.map((id) => (
        <li key={id}>
          <button
            type="button"
            onClick={() => onSelect(id)}
            aria-pressed={recipient === id}
            title={`Message ${id} privately`}
            className={cn(row, recipient === id && "bg-hover")}
          >
            <Avatar id={id} />
            <span className="min-w-0 flex-1">
              <span className="block truncate font-mono text-sm">{shortId(id)}</span>
              <span className="block text-xs text-muted">Private message</span>
            </span>
          </button>
        </li>
      ))}
      {others.length === 0 && (
        <li className="px-3 py-3 text-sm leading-relaxed text-muted">
          You&apos;re the only one here. Open the app in a second window to
          start a conversation.
        </li>
      )}
    </ul>
  );
}
