"use client";

import { useEffect, useRef, useState } from "react";
import { useChat } from "@/providers/websocketProvider";
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
import Link from "next/link";
import { WebSocketStateEnum } from "@/providers/websocketProvider";
import { MessageTypeEnum } from "@/providers/websocketProvider";
// import { Separator } from "@/components/ui/separator";

export default function Page() {
  const { status: websocketStatus, authState, messages, clientId, connectedClients, sendMessage } = useChat();

  const [input, setInput] = useState("");
  const [selectedRecipient, setSelectedRecipient] = useState("all");
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new message arrives
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleSend = () => {
    const trimmedInput = input.trim();
    if (trimmedInput && websocketStatus === WebSocketStateEnum.OPEN) {
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

  return (
    <div className="flex h-screen flex-col gap-4 bg-slate-950 p-4">
      {/* Header */}
      <Card className="border-slate-800 bg-slate-900">
        <CardHeader>
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <CardTitle className="text-2xl">WebSocket Chat</CardTitle>
              <div className="text-sm text-slate-400">
                <span className="font-medium">Your ID:</span> <span className="text-slate-300">{clientId || "..."}</span>
              </div>
            </div>
            <div className="flex items-center gap-6">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-slate-400">Connection:</span>
                <Badge
                  variant={
                    websocketStatus === WebSocketStateEnum.OPEN
                      ? "default"
                      : websocketStatus === WebSocketStateEnum.CLOSED
                        ? "destructive"
                        : "secondary"
                  }
                >
                  {websocketStatus}
                </Badge>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-slate-400">Auth:</span>
                <Badge
                  variant={
                    authState === "authenticated"
                      ? "default"
                      : authState === "unauthenticated"
                        ? "destructive"
                        : "secondary"
                  }
                >
                  {authState}
                </Badge>
              </div>
            </div>
            <div className="text-sm text-slate-400">
              <span className="font-medium">{connectedClients.length} users online</span>
            </div>
          </div>
        </CardHeader>
      </Card>

      {/* Error Display */}
      {/* {error && (
        <Card className="border-red-200 bg-red-50">
          <CardContent className="pt-6">
            <p className="text-sm text-red-700">⚠️ {error}</p>
          </CardContent>
        </Card>
      )} */}

      {/* Recipient Selector */}
      <Card className="border-slate-800 bg-slate-900">
        <CardContent className="pt-6">
          <div className="flex items-center gap-2">
            <label className="text-sm font-medium text-slate-300">Send to:</label>
            <Select value={selectedRecipient} onValueChange={setSelectedRecipient}>
              <SelectTrigger className="w-64 border-slate-700 bg-slate-800 text-slate-200">
                <SelectValue />
              </SelectTrigger>
              <SelectContent className="border-slate-700 bg-slate-900">
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
      <Card className="flex flex-1 flex-col overflow-hidden border-slate-800 bg-slate-900">
        <CardHeader>
          <CardTitle className="text-base text-slate-200">Message History</CardTitle>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden p-0">
          <ScrollArea className="h-full w-full">
            <div className="flex flex-col gap-2 p-4">
              {messages.length === 0 ? (
                <p className="text-sm text-slate-500">No messages yet...</p>
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
                        className={`text-xs font-mono leading-relaxed ${msg.type === MessageTypeEnum.HEARTBEAT
                            ? "text-slate-600"
                            : msg.type === MessageTypeEnum.JOIN
                              ? "text-green-400"
                              : msg.type === MessageTypeEnum.LEAVE
                                ? "text-orange-400"
                                : isPartOfPrivate
                                  ? "italic text-purple-400"
                                  : "text-slate-300"
                          }`}
                      >
                        <span className="text-slate-500">[{msg.timestamp}]</span>{" "}
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
      <Card className="border-slate-800 bg-slate-900">
        <CardContent className="pt-6">
          <div className="flex gap-2">
            <Input
              type="text"
              placeholder="Type a message..."
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyPress={handleKeyPress}
              disabled={websocketStatus !== WebSocketStateEnum.OPEN}
              className="flex-1 border-slate-700 bg-slate-800 text-slate-200 placeholder-slate-500"
            />
            <Button
              onClick={handleSend}
              disabled={!input.trim() || websocketStatus !== WebSocketStateEnum.OPEN}
            >
              Send
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

