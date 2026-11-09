import { Message } from "@/providers/websocketProvider";

/**
 * Formats a chat message for display. Only chat messages are stored in the
 * message list; presence and heartbeat traffic is handled by the provider.
 */
export function formatMessageDisplay(
  msg: Message,
  currentClientId: string
): string {
  const sender = msg.clientId ?? "?";
  const text = msg.payload?.text ?? "";
  const recipientId = msg.payload?.recipientId;
  const isPrivate =
    !!recipientId &&
    recipientId !== "all" &&
    (msg.clientId === currentClientId || recipientId === currentClientId);

  return isPrivate ? `[${sender}] (private): ${text}` : `[${sender}]: ${text}`;
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
