import {
  WebSocketStateEnum,
  type AuthState,
  type WebSocketState,
} from "@/providers/websocketProvider";
import { cn } from "@/lib/utils";

type View = { label: string; detail: string; dot: string; pulse: boolean };

export function describeConnection(
  status: WebSocketState,
  authState: AuthState
): View {
  if (status === WebSocketStateEnum.OPEN && authState === "authenticated") {
    return { label: "Connected", detail: "Authenticated", dot: "bg-ok", pulse: false };
  }
  if (status === WebSocketStateEnum.OPEN) {
    return { label: "Signing in", detail: "Waiting for the server", dot: "bg-signal", pulse: true };
  }
  if (status === WebSocketStateEnum.RECONNECTING) {
    return { label: "Reconnecting", detail: "Retrying with backoff", dot: "bg-private", pulse: true };
  }
  if (status === WebSocketStateEnum.CONNECTING) {
    return { label: "Connecting", detail: "Opening the socket", dot: "bg-signal", pulse: true };
  }
  return { label: "Disconnected", detail: "Not connected to the server", dot: "bg-bad", pulse: false };
}

export function ConnectionStatus({
  status,
  authState,
}: {
  status: WebSocketState;
  authState: AuthState;
}) {
  const v = describeConnection(status, authState);
  return (
    <div role="status" className="flex items-center gap-3">
      <span className="relative flex size-2.5">
        {v.pulse && (
          <span
            className={cn(
              "absolute inline-flex size-full rounded-full opacity-60 motion-safe:animate-ping",
              v.dot
            )}
          />
        )}
        <span className={cn("relative inline-flex size-2.5 rounded-full", v.dot)} />
      </span>
      <div className="leading-tight">
        <p className="text-sm font-medium">{v.label}</p>
        <p className="text-xs text-muted">{v.detail}</p>
      </div>
    </div>
  );
}
