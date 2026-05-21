"use client";

import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";

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
  JOIN: "join",
  LEAVE: "leave",
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
    clientList?: string[];
    success?: boolean; // For auth response
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
  const reconnectTimeout = useRef<NodeJS.Timeout | null>(null);
  const manuallyClosed = useRef(false);
  const storeRef = useRef(store);

  // -----------------------
  // CONNECT LOGIC
  // -----------------------
  const connect = () => {
    setStore((prev) => ({ ...prev, status: retryCount.current > 0 ? WebSocketStateEnum.RECONNECTING : WebSocketStateEnum.CONNECTING }));

    const socket = new WebSocket("ws://localhost:8080/ws");
    ws.current = socket;

    socket.onopen = () => {
      setStore((prev) => ({ ...prev, status: "open" }));
      retryCount.current = 0;

      //
      const authPayload = {
        type: MessageTypeEnum.AUTH,
        payload: {
          //token: localStorage.getItem("token") || "secret-token", // Replace with real token retrieval
          token: sessionStorage.getItem("token") || "secret-token", // Replace with real token retrieval
          clientType: storeRef.current.clientType ?? CLIENT_TYPE.BROWSER,
          clientId: storeRef.current.clientId || undefined,
        },
      };
      setStore((prev) => ({ ...prev, authState: AuthStateEnum.AUTHENTICATING }));
      ws.current?.send(JSON.stringify(authPayload));
    };

    socket.onmessage = (event) => {
      const data: Message = JSON.parse(event.data);

      // 0. HEARTBEAT HANDLING (NEW)
      if (data.type === MessageTypeEnum.HEARTBEAT) {
        // For now, we dont do anything on heartbeat, but we could update a last-seen timestamp or user count if we wanted
        return; // don't treat as chat message
      }

      // -------------------------
      // 1. CLIENT LIST UPDATES
      // -------------------------
      if (
        data.type === MessageTypeEnum.JOIN ||
        data.type === MessageTypeEnum.LEAVE
      ) {
        if (data.payload?.clientList) {
          setStore((prev) => ({ ...prev, connectedClients: data.payload?.clientList ?? [] }));
        }
      }

      // -------------------------
      // 2. CLIENT ID INITIALIZATION
      // -------------------------
      if (!storeRef.current.clientId && data.clientId) {
        setStore((prev) => ({ ...prev, clientId: data.clientId ?? "" }));
        //localStorage.setItem("websocket-clientid", data.clientId);
        sessionStorage.setItem("websocket-clientid", data.clientId);
      }

      // -------------------------
      // 3. AUTH RESPONSE HANDLING (NEW)
      // -------------------------
      if (data.type === MessageTypeEnum.AUTH_RESPONSE) {
        setStore((prev) => ({
          ...prev,
          status: data.payload?.success ? WebSocketStateEnum.OPEN : WebSocketStateEnum.CLOSED,
          authState: data.payload?.success ? AuthStateEnum.AUTHENTICATED : AuthStateEnum.UNAUTHENTICATED,
        }));

        // if auth failed, dont keep retrying
        if (!data.payload?.success) {
          manuallyClosed.current = true;
          ws.current?.close();
        }

        return; // IMPORTANT: don't treat as chat message
      }

      // -------------------------
      // 4. CHAT MESSAGES ONLY
      // -------------------------
      if (data.type ===  MessageTypeEnum.MESSAGE) {
        setStore((prev) => ({
          ...prev,
          messages: [...prev.messages, data],
        }));
      }

      //setStore((prev) => ({ ...prev, messages: [...prev.messages, data] }));
    };

    socket.onclose = () => {
      setStore((prev) => ({ ...prev, status: WebSocketStateEnum.CLOSED }));

      if (manuallyClosed.current) return;

      const delay = Math.min(
        1000 * Math.pow(2, retryCount.current),
        10000
      );

      retryCount.current++;

      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }

      reconnectTimeout.current = setTimeout(() => {
        connect();
      }, delay);
    };

    socket.onerror = () => {
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
    //const stored = localStorage.getItem("websocket-clientid");
    const stored = sessionStorage.getItem("websocket-clientid");
    if (stored) setStore((prev) => ({ ...prev, clientId: stored }));

    connect();

    return () => {
      manuallyClosed.current = true;

      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }

      ws.current?.close();
    };
  }, []);

  // Debugging: Log when provider mounts/unmounts
  useEffect(() => {
    console.log("websocketProvider mounted");

    return () => {
      console.log("websocketProvider unmounted");
    };
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