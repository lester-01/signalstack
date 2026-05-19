//import { Message } from "@/hooks/useWebSocket";
import { Message } from "@/providers/websocketProvider";

/**
 * Checks if a message is private (not visible to all clients)
 */
export function isPrivateMessage(
  msg: Message,
  currentClientId: string
): boolean {
  const recipientId = msg.payload?.recipientId;
  if (!recipientId || recipientId === "all") {
    return false;
  }
  // Private if current client is sender or recipient
  return msg.clientId === currentClientId || recipientId === currentClientId;
}

/**
 * Formats a message for display based on its type
 */
export function formatMessageDisplay(
  msg: Message,
  currentClientId: string
): string {
  //const senderInitial = msg.clientId ? msg.clientId.slice(0, 1) : "?";
  const senderInitial = msg.clientId ? msg.clientId : "?";
  const userCount = msg.payload?.userCount ?? 0;
  const isPrivate =
    msg.payload?.recipientId &&
    msg.payload.recipientId !== "all" &&
    (msg.clientId === currentClientId ||
      msg.payload.recipientId === currentClientId);

  switch (msg.type) {
    case "message": {
      const text = msg.payload?.text || "";
      if (isPrivate) {
        return `[${senderInitial}] (private): ${text}`;
      }
      return `[${senderInitial}]: ${text}`;
    }

    case "join": {
      return `→ User ${senderInitial} joined (${userCount} online)`;
    }

    case "leave": {
      return `← User ${senderInitial} left (${userCount} online)`;
    }

    case "heartbeat": {
      return `💓 [Heartbeat] ${userCount} users online`;
    }

    default:
      return "[Unknown message type]";
  }
}

/**
 * Gets display label for a client ID (shows your own ID with "(you)" suffix)
 */
export function getClientLabel(clientId: string, currentClientId: string): string {
  if (clientId === currentClientId) {
    return `${clientId} (you)`;
  }
  return clientId;
}
