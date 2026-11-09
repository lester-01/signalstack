"use client";

import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";

// Configuration comes from the environment (see .env.example). The WebSocket
// URL falls back to the local dev backend; the auth token has no fallback.
const WS_URL = process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080/ws";

const STORAGE_KEYS = {
  authToken: "token", // optional override, e.g. set by a future login screen
  clientId: "websocket-clientid",
  resumeToken: "websocket-resume-token",
} as const;

const RECONNECT_BASE_MS = 1000;
const RECONNECT_MAX_MS = 10000;
const MAX_MESSAGES = 1000; // keep only the most recent messages in memory

function getAuthToken(): string {
  return (
    sessionStorage.getItem(STORAGE_KEYS.authToken) ??
    process.env.NEXT_PUBLIC_AUTH_TOKEN ??
    ""
  );
}

export const CLIENT_TYPE = {
  BROWSER: "browser",
  ANDROID: "android_client",
  LINUX: "linux_client",
} as const;

export type ClientType =
  typeof CLIENT_TYPE[keyof typeof CLIENT_TYPE];

export const MessageTypeEnum = {
  MESSAGE: "message",
  HEARTBEAT: "heartbeat",
  PRESENCE: "presence",
  AUTH: "auth",
  AUTH_RESPONSE: "auth_response"
} as const;

export type MessageType = typeof MessageTypeEnum[keyof typeof MessageTypeEnum];

export type Message = {
  type: MessageType;
  clientId?: string;
  timestamp: string;
  payload?: {
    text?: string;
    recipientId?: string;
    userCount?: number;
    clientList?: string[]; // auth_response: everyone online, including you
    joined?: string[]; // presence: clients that came online
    left?: string[]; // presence: clients that went offline
    success?: boolean; // auth_response
    reason?: string; // auth_response failure reason
    resumeToken?: string; // auth_response success: lets this client reclaim its ID
  };
};

export const WebSocketStateEnum = {
  CONNECTING: "connecting",
  OPEN: "open",
  CLOSED: "closed",
  RECONNECTING: "reconnecting"
} as const;

export type WebSocketState = typeof WebSocketStateEnum[keyof typeof WebSocketStateEnum];

const AuthStateEnum = {
  UNAUTHENTICATED: "unauthenticated",
  AUTHENTICATING: "authenticating",
  AUTHENTICATED: "authenticated"
} as const;
export type AuthState = typeof AuthStateEnum[keyof typeof AuthStateEnum];

type Store = {
  status: WebSocketState;
  clientId: string;
  messages: Message[];
  connectedClients: string[];
  clientType: ClientType;
  authState: AuthState;
};

type ContextValue = Store & {
  sendMessage: (text: string, recipientId?: string) => void;
};

const WebSocketContext = createContext<ContextValue | null>(null);

export function WebSocketProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  // -----------------------
  // STATE (STORE)
  // -----------------------
  const [store, setStore] = useState<Store>({
    status: WebSocketStateEnum.CONNECTING,
    clientId: "",
    messages: [],
    connectedClients: [],
    clientType: CLIENT_TYPE.BROWSER,
    authState: AuthStateEnum.UNAUTHENTICATED,
  });

  // -----------------------
  // SOCKET REFS (NO RERENDER)
  // -----------------------
  const ws = useRef<WebSocket | null>(null);
  const retryCount = useRef(0);
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const manuallyClosed = useRef(false);
  const storeRef = useRef(store);

  // -----------------------
  // CONNECT LOGIC
  // -----------------------
  const connect = () => {
    setStore((prev) => ({
      ...prev,
      status:
        retryCount.current > 0
          ? WebSocketStateEnum.RECONNECTING
          : WebSocketStateEnum.CONNECTING,
    }));

    const socket = new WebSocket(WS_URL);
    ws.current = socket;

    // Handlers ignore events from a socket that is no longer the current one
    // (e.g. a socket closed by a React remount), so it cannot trigger a
    // reconnect or overwrite state.
    socket.onopen = () => {
      if (ws.current !== socket) return;

      setStore((prev) => ({
        ...prev,
        status: WebSocketStateEnum.OPEN,
        authState: AuthStateEnum.AUTHENTICATING,
      }));

      // clientId + resumeToken let the server restore this client's identity.
      // The server ignores them unless the token is valid.
      socket.send(
        JSON.stringify({
          type: MessageTypeEnum.AUTH,
          payload: {
            token: getAuthToken(),
            clientType: storeRef.current.clientType ?? CLIENT_TYPE.BROWSER,
            clientId: sessionStorage.getItem(STORAGE_KEYS.clientId) ?? undefined,
            resumeToken:
              sessionStorage.getItem(STORAGE_KEYS.resumeToken) ?? undefined,
          },
        })
      );
    };

    socket.onmessage = (event) => {
      if (ws.current !== socket) return;

      let data: Message;
      try {
        data = JSON.parse(event.data);
      } catch {
        return; // ignore malformed frames
      }

      switch (data.type) {
        case MessageTypeEnum.HEARTBEAT:
          return;

        case MessageTypeEnum.AUTH_RESPONSE: {
          const ok = data.payload?.success === true;

          if (ok && data.clientId) {
            // The server is the source of truth for our ID. It may differ from
            // the one we sent if the resume token was missing or invalid.
            sessionStorage.setItem(STORAGE_KEYS.clientId, data.clientId);
            if (data.payload?.resumeToken) {
              sessionStorage.setItem(
                STORAGE_KEYS.resumeToken,
                data.payload.resumeToken
              );
            }
            retryCount.current = 0;
          }

          const list = data.payload?.clientList;
          setStore((prev) => ({
            ...prev,
            clientId: ok && data.clientId ? data.clientId : prev.clientId,
            // The welcome message carries the full list; presence messages
            // after it are deltas.
            connectedClients: ok && list ? list : prev.connectedClients,
            status: ok ? WebSocketStateEnum.OPEN : WebSocketStateEnum.CLOSED,
            authState: ok
              ? AuthStateEnum.AUTHENTICATED
              : AuthStateEnum.UNAUTHENTICATED,
          }));

          // Don't keep retrying with credentials the server rejected.
          if (!ok) {
            manuallyClosed.current = true;
            socket.close();
          }
          return;
        }

        case MessageTypeEnum.PRESENCE: {
          const joined = data.payload?.joined ?? [];
          const left = new Set(data.payload?.left ?? []);
          setStore((prev) => {
            const next = prev.connectedClients.filter((id) => !left.has(id));
            for (const id of joined) {
              if (!next.includes(id)) next.push(id);
            }
            return { ...prev, connectedClients: next };
          });
          return;
        }

        case MessageTypeEnum.MESSAGE:
          setStore((prev) => ({
            ...prev,
            messages: [...prev.messages, data].slice(-MAX_MESSAGES),
          }));
          return;
      }
    };

    socket.onclose = () => {
      if (ws.current !== socket) return;

      setStore((prev) => ({
        ...prev,
        status: WebSocketStateEnum.CLOSED,
        authState: AuthStateEnum.UNAUTHENTICATED,
      }));

      if (manuallyClosed.current) return;

      // Exponential backoff with jitter, so many clients that lost the same
      // server do not all reconnect at the same instant.
      const backoff = Math.min(
        RECONNECT_BASE_MS * 2 ** retryCount.current,
        RECONNECT_MAX_MS
      );
      const delay = backoff / 2 + Math.random() * (backoff / 2);

      retryCount.current++;

      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }

      reconnectTimeout.current = setTimeout(connect, delay);
    };

    socket.onerror = () => {
      if (ws.current !== socket) return;
      setStore((prev) => ({ ...prev, status: WebSocketStateEnum.CLOSED }));
    };
  };

  // -----------------------
  // ACTIONS (UI → WS)
  // -----------------------
  const sendMessage = (text: string, recipientId = "all") => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) return;

    ws.current.send(
      JSON.stringify({
        type: MessageTypeEnum.MESSAGE,
        timestamp: "",
        payload: { text, recipientId },
      })
    );
  };

  // -----------------------
  // SYNC STORE REF (FOR WS CALLBACKs)
  // -----------------------
  useEffect(() => {
    storeRef.current = store;
  }, [store]);

  // -----------------------
  // INIT / CLEANUP
  // -----------------------
  useEffect(() => {
    // Reset on every mount: React StrictMode runs this effect, its cleanup and
    // the effect again, and the cleanup below sets this flag.
    manuallyClosed.current = false;

    connect();

    return () => {
      manuallyClosed.current = true;

      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }

      const socket = ws.current;
      ws.current = null; // marks the socket stale so its handlers do nothing
      socket?.close();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // -----------------------
  // CONTEXT VALUE
  // -----------------------
  const value: ContextValue = {
    ...store,
    sendMessage,
  };

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
}

// -----------------------
// CONSUMER HOOK
// -----------------------
export function useChat() {
  const ctx = useContext(WebSocketContext);
  if (!ctx) {
    throw new Error("useChat must be used inside WebSocketProvider");
  }
  return ctx;
}
