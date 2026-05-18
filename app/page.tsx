"use client";

import { useEffect, useRef, useState } from "react";
import { useWebSocket } from "@/hooks/useWebSocket";
import { formatMessageDisplay, getClientLabel } from "@/lib/utils/timestamp";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";

export default function Page() {
  const { status, messages, clientId, connectedClients, error, sendMessage } =
    useWebSocket("ws://localhost:8080/ws");

  const [input, setInput] = useState("");
  const [selectedRecipient, setSelectedRecipient] = useState("all");
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new message arrives
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleSend = () => {
    const trimmedInput = input.trim();
    if (trimmedInput && status === "open") {
      sendMessage(trimmedInput, selectedRecipient);
      setInput("");
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  // Get user count from latest heartbeat or join/leave
  const userCount = messages
    .reverse()
    .find((m) =>
      ["heartbeat", "join", "leave"].includes(m.type)
    )?.payload?.userCount ?? connectedClients.length;

  return (
    <div className="flex h-screen flex-col gap-4 bg-slate-50 p-4">
      {/* Header */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <CardTitle>WebSocket Chat</CardTitle>
              <Badge
                variant={
                  status === "open"
                    ? "default"
                    : status === "closed"
                      ? "destructive"
                      : "secondary"
                }
              >
                {status}
              </Badge>
            </div>
            <div className="text-sm">
              <span className="font-medium">Your ID:</span> {clientId || "..."}
            </div>
          </div>
          <div className="mt-2 text-sm text-slate-600">
            <span className="font-medium">{userCount} users online</span>
          </div>
        </CardHeader>
      </Card>

      {/* Error Display */}
      {error && (
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-sm text-red-700">⚠️ {error}</p>
          </CardContent>
        </Card>
      )}

      {/* Recipient Selector */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium">Send to:</label>
            <Select value={selectedRecipient} onValueChange={setSelectedRecipient}>
              <SelectTrigger className="w-64">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Everyone</SelectItem>
                {connectedClients
                  .filter((id) => id !== clientId)
                  .map((id) => (
                    <SelectItem key={id} value={id}>
                      {getClientLabel(id, clientId)}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Messages Display */}
      <Card className="flex flex-1 flex-col overflow-hidden">
        <CardHeader>
          <CardTitle className="text-base">Message History</CardTitle>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden p-0">
          <ScrollArea className="h-full w-full">
            <div className="flex flex-col gap-2 p-4">
              {messages.length === 0 ? (
                <p className="text-sm text-slate-400">No messages yet...</p>
              ) : (
                messages.map((msg, idx) => {
                  const displayText = formatMessageDisplay(msg, clientId);
                  const isPrivate =
                    msg.payload?.recipientId &&
                    msg.payload.recipientId !== "all";
                  const isPartOfPrivate =
                    isPrivate &&
                    (msg.clientId === clientId ||
                      msg.payload?.recipientId === clientId);

                  return (
                    <div key={idx}>
                      <div
                        className={`text-xs font-mono leading-relaxed ${
                          msg.type === "heartbeat"
                            ? "text-slate-400"
                            : msg.type === "join"
                              ? "text-green-600"
                              : msg.type === "leave"
                                ? "text-orange-600"
                                : isPartOfPrivate
                                  ? "italic text-purple-600"
                                  : "text-slate-700"
                        }`}
                      >
                        <span className="text-slate-400">[{msg.timestamp}]</span>{" "}
                        {displayText}
                      </div>
                    </div>
                  );
                })
              )}
              <div ref={messagesEndRef} />
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      {/* Input Section */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex gap-2">
            <Input
              type="text"
              placeholder="Type a message..."
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyPress={handleKeyPress}
              disabled={status !== "open"}
              className="flex-1"
            />
            <Button
              onClick={handleSend}
              disabled={!input.trim() || status !== "open"}
            >
              Send
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

