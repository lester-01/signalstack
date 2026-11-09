"use client";

import { useEffect, useRef } from "react";
import type { Message } from "@/providers/websocketProvider";
import { idColor, idTint, shortId } from "@/lib/identity";
import { cn } from "@/lib/utils";
import { Avatar } from "./avatar";
import { Lock } from "./icons";

type Run = { sender: string; to: string | null; items: Message[] };

/** Consecutive messages from one sender to the same audience form a run. */
function toRuns(messages: Message[]): Run[] {
  const runs: Run[] = [];
  for (const m of messages) {
    const sender = m.clientId ?? "unknown";
    const raw = m.payload?.recipientId;
    const to = raw && raw !== "all" ? raw : null;
    const last = runs[runs.length - 1];
    if (last && last.sender === sender && last.to === to) last.items.push(m);
    else runs.push({ sender, to, items: [m] });
  }
  return runs;
}

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
      <div className="grid flex-1 place-items-center px-8 text-center">
        <div>
          <p className="text-lg font-semibold tracking-tight">Say something</p>
          <p className="mt-1 max-w-64 text-sm leading-relaxed text-muted">
            Messages go to everyone. Pick a person from the list to talk to them
            privately.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div
      role="log"
      aria-live="polite"
      aria-label="Messages"
      className="flex-1 overflow-y-auto px-4 pt-4 pb-28 sm:px-8"
    >
      <div className="mx-auto flex max-w-2xl flex-col gap-5">
        {toRuns(messages).map((run, i) => {
          const mine = run.sender === selfId;
          const isPrivate = run.to !== null;
          const first = run.items[0];

          return (
            <section
              key={`${first.timestamp}-${i}`}
              className={cn("flex items-end gap-2.5", mine && "flex-row-reverse")}
            >
              {!mine && <Avatar id={run.sender} className="size-8 text-[10px]" />}
              <div
                className={cn(
                  "flex min-w-0 max-w-[82%] flex-col gap-1",
                  mine && "items-end"
                )}
              >
                <p className="flex items-center gap-2 px-1 text-xs text-muted">
                  {!mine && (
                    <span
                      title={run.sender}
                      className="font-mono font-semibold"
                      style={{ color: idColor(run.sender) }}
                    >
                      {shortId(run.sender)}
                    </span>
                  )}
                  {isPrivate && (
                    <span className="inline-flex items-center gap-1 font-medium text-private">
                      <Lock />
                      {mine ? `Only ${shortId(run.to!)}` : "Only you"}
                    </span>
                  )}
                  <time className="font-mono tabular-nums">{first.timestamp}</time>
                </p>
                {run.items.map((m, j) => (
                  <p
                    key={j}
                    className={cn(
                      "wrap-anywhere whitespace-pre-wrap rounded-3xl px-4 py-2.5 text-[15px] leading-snug",
                      mine ? "bg-ink text-paper" : "text-ink",
                      isPrivate && "outline-2 -outline-offset-2 outline-private/70"
                    )}
                    style={mine ? undefined : { backgroundColor: idTint(run.sender) }}
                  >
                    {m.payload?.text}
                  </p>
                ))}
              </div>
            </section>
          );
        })}
      </div>
      <div ref={endRef} />
    </div>
  );
}
