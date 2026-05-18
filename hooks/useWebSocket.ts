"use client";

import { useEffect, useRef, useState } from "react";

export type Status = "connecting" | "open" | "closed" | "reconnecting";

export interface Message {
  type: "message" | "heartbeat" | "join" | "leave";
  clientId?: string;
  timestamp: string;
  payload?: {
    text?: string;
    recipientId?: string;
    userCount?: number;
    clientList?: string[];
  };
}

export function useWebSocket(url: string) {
  const ws = useRef<WebSocket | null>(null);
  const retryCount = useRef(0);
  const [status, setStatus] = useState<Status>("connecting");
  const [messages, setMessages] = useState<Message[]>([]);
  const [clientId, setClientId] = useState<string>("");
  const [connectedClients, setConnectedClients] = useState<string[]>([]);
  const [error, setError] = useState<string>("");

  const connect = () => {
    setStatus(retryCount.current > 0 ? "reconnecting" : "connecting");
    setError("");

    const socket = new WebSocket(url);
    ws.current = socket;

    socket.onopen = () => {
      setStatus("open");
      retryCount.current = 0;
      console.log("WebSocket connected");
    };

    socket.onmessage = (event) => {
      try {
        const data: Message = JSON.parse(event.data);

        // Extract client list and update state
        if (
          data.type === "join" ||
          data.type === "leave" ||
          data.type === "heartbeat"
        ) {
          if (data.payload?.clientList) {
            setConnectedClients(data.payload.clientList);
          }
        }

        // Set client ID from first message (should be join)
        if (!clientId && data.clientId) {
          setClientId(data.clientId);
          localStorage.setItem("websocket-clientid", data.clientId);
        }

        // Add message to history
        setMessages((prev) => [...prev, data]);
      } catch (err) {
        console.error("Error parsing message:", err);
      }
    };

    socket.onerror = () => {
      setError("WebSocket connection error");
      console.error("WebSocket error");
    };

    socket.onclose = () => {
      setStatus("closed");

      // Calculate exponential backoff: 1s, 2s, 4s, 8s, max 10s
      const delay = Math.min(1000 * Math.pow(2, retryCount.current), 10000);
      retryCount.current += 1;

      console.log(`Reconnecting in ${delay}ms...`);
      setTimeout(() => connect(), delay);
    };
  };

  useEffect(() => {
    // Load stored client ID if available
    const storedClientId = localStorage.getItem("websocket-clientid");
    if (storedClientId) {
      setClientId(storedClientId);
    }

    connect();

    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, []);

  const sendMessage = (text: string, recipientId: string = "all") => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      console.error("WebSocket not open");
      return;
    }

    const msg: Message = {
      type: "message",
      timestamp: "", // Server will add timestamp
      payload: {
        text,
        recipientId,
      },
    };

    ws.current.send(JSON.stringify(msg));
  };

  return { status, messages, clientId, connectedClients, error, sendMessage };
}
