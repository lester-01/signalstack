"use client";

import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";

type Message = {
  type: "message" | "heartbeat" | "join" | "leave";
  clientId?: string;
  timestamp: string;
  payload?: {
    text?: string;
    recipientId?: string;
    userCount?: number;
    clientList?: string[];
  };
};

type Store = {
  status: "connecting" | "open" | "closed" | "reconnecting";
  clientId: string;
  messages: Message[];
  connectedClients: string[];
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
  const [status, setStatus] =
    useState<Store["status"]>("connecting");
  const [clientId, setClientId] = useState("");
  const [messages, setMessages] = useState<Message[]>([]);
  const [connectedClients, setConnectedClients] = useState<string[]>([]);

  // -----------------------
  // SOCKET REFS (NO RERENDER)
  // -----------------------
  const ws = useRef<WebSocket | null>(null);
  const retryCount = useRef(0);
  const reconnectTimeout = useRef<NodeJS.Timeout | null>(null);
  const manuallyClosed = useRef(false);

  // -----------------------
  // CONNECT LOGIC
  // -----------------------
  const connect = () => {
    setStatus(retryCount.current > 0 ? "reconnecting" : "connecting");

    const socket = new WebSocket("ws://localhost:8080/ws");
    ws.current = socket;

    socket.onopen = () => {
      setStatus("open");
      retryCount.current = 0;
    };

    socket.onmessage = (event) => {
      const data: Message = JSON.parse(event.data);

      // client list updates
      if (
        data.type === "join" ||
        data.type === "leave" ||
        data.type === "heartbeat"
      ) {
        if (data.payload?.clientList) {
          setConnectedClients(data.payload.clientList);
        }
      }

      // clientId init
      if (!clientId && data.clientId) {
        setClientId(data.clientId);
        localStorage.setItem("websocket-clientid", data.clientId);
      }

      setMessages((prev) => [...prev, data]);
    };

    socket.onclose = () => {
        setStatus("closed");

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
      setStatus("closed");
    };
  };

  // -----------------------
  // ACTIONS (UI → WS)
  // -----------------------
  const sendMessage = (text: string, recipientId = "all") => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) return;

    ws.current.send(
      JSON.stringify({
        type: "message",
        timestamp: "",
        payload: { text, recipientId },
      })
    );
  };

  // -----------------------
  // INIT / CLEANUP
  // -----------------------
  useEffect(() => {
    const stored = localStorage.getItem("websocket-clientid");
    if (stored) setClientId(stored);

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
  console.log("Provider mounted");

  return () => {
    console.log("Provider unmounted");
  };
}, []);

  // -----------------------
  // CONTEXT VALUE
  // -----------------------
  const value: ContextValue = {
    status,
    clientId,
    messages,
    connectedClients,
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