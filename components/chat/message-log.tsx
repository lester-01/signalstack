"use client";

import { useEffect, useRef } from "react";
import type { Message } from "@/providers/websocketProvider";
import { idColor, shortId } from "@/lib/identity";
import { cn } from "@/lib/utils";
import { Avatar } from "./avatar";

export function MessageLog({
  messages,
  selfId,
}: {
  messages: Message[];
  selfId: string;
}) {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ block: "end" });
  }, [messages]);

  if (messages.length === 0) {
    return (
      <div className="grid flex-1 place-items-center px-6 text-center">
        <p className="max-w-xs text-sm text-muted">
          No messages yet. Write one below, or pick someone from the list to
          message them privately.
        </p>
      </div>
    );
  }

  return (
    <div
      role="log"
      aria-live="polite"
      aria-label="Messages"
      className="flex-1 overflow-y-auto px-4 py-4 sm:px-6"
    >
      <ol className="mx-auto flex max-w-2xl flex-col gap-3">
        {messages.map((msg, i) => {
          const sender = msg.clientId ?? "unknown";
          const mine = sender === selfId;
          const to = msg.payload?.recipientId;
          const isPrivate = !!to && to !== "all";

          return (
            <li
              key={`${msg.timestamp}-${i}`}
              className={cn("flex items-end gap-2.5", mine && "flex-row-reverse")}
            >
              {!mine && <Avatar id={sender} className="size-7 text-[10px]" />}
              <div className={cn("flex min-w-0 max-w-[85%] flex-col gap-1", mine && "items-end")}>
                <p className="flex items-baseline gap-2 text-xs text-muted">
                  <span
                    title={mine ? undefined : sender}
                    className="font-mono font-medium"
                    style={{ color: mine ? undefined : idColor(sender) }}
                  >
                    {mine ? "You" : shortId(sender)}
                  </span>
                  {isPrivate && (
                    <span className="rounded bg-private/15 px-1.5 py-px font-medium text-private">
                      {mine ? `private to ${shortId(to)}` : "private to you"}
                    </span>
                  )}
                  <time className="font-mono tabular-nums">{msg.timestamp}</time>
                </p>
                <p
                  className={cn(
                    "wrap-anywhere whitespace-pre-wrap rounded-2xl px-3.5 py-2 text-[15px] leading-snug",
                    mine
                      ? "rounded-br-md bg-signal text-white"
                      : "rounded-bl-md bg-bubble",
                    isPrivate && mine && "bg-private",
                    isPrivate && !mine && "bg-private/12 ring-1 ring-private/30"
                  )}
                >
                  {msg.payload?.text}
                </p>
              </div>
            </li>
          );
        })}
      </ol>
      <div ref={endRef} />
    </div>
  );
}
