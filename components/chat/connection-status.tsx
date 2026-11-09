import {
  WebSocketStateEnum,
  type AuthState,
  type WebSocketState,
} from "@/providers/websocketProvider";
import { cn } from "@/lib/utils";

type View = { label: string; dot: string; pulse: boolean };

export function describeConnection(
  status: WebSocketState,
  authState: AuthState
): View {
  if (status === WebSocketStateEnum.OPEN && authState === "authenticated") {
    return { label: "Live", dot: "bg-ok", pulse: false };
  }
  if (status === WebSocketStateEnum.OPEN) {
    return { label: "Signing in", dot: "bg-signal", pulse: true };
  }
  if (status === WebSocketStateEnum.RECONNECTING) {
    return { label: "Reconnecting", dot: "bg-private", pulse: true };
  }
  if (status === WebSocketStateEnum.CONNECTING) {
    return { label: "Connecting", dot: "bg-signal", pulse: true };
  }
  return { label: "Offline", dot: "bg-bad", pulse: false };
}

export function ConnectionPill({
  status,
  authState,
}: {
  status: WebSocketState;
  authState: AuthState;
}) {
  const v = describeConnection(status, authState);
  return (
    <span
      role="status"
      className="inline-flex items-center gap-2 rounded-full bg-hover px-2.5 py-1 text-xs font-medium"
    >
      <span className="relative flex size-2">
        {v.pulse && (
          <span
            className={cn(
              "absolute inline-flex size-full rounded-full opacity-60 motion-safe:animate-ping",
              v.dot
            )}
          />
        )}
        <span className={cn("relative inline-flex size-2 rounded-full", v.dot)} />
      </span>
      {v.label}
    </span>
  );
}
