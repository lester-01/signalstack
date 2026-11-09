"use client";

import { useState } from "react";
import {
  useChat,
  WebSocketStateEnum,
} from "@/providers/websocketProvider";
import { Avatar } from "@/components/chat/avatar";
import { Composer } from "@/components/chat/composer";
import { ConnectionStatus } from "@/components/chat/connection-status";
import { MessageLog } from "@/components/chat/message-log";
import { shortId } from "@/lib/identity";
import { PresenceList } from "@/components/chat/presence-list";

export default function Page() {
  const { status, authState, messages, clientId, connectedClients, sendMessage } =
    useChat();
  const [picked, setPicked] = useState("all");

  // If the person we were messaging goes offline, fall back to everyone.
  const recipient =
    picked === "all" || connectedClients.includes(picked) ? picked : "all";
  const ready =
    status === WebSocketStateEnum.OPEN && authState === "authenticated";
  const targetLabel = recipient === "all" ? "everyone" : recipient;

  return (
    <div className="flex h-dvh flex-col md:flex-row">
      <aside className="flex max-h-[38dvh] shrink-0 flex-col border-b border-line bg-panel md:max-h-none md:w-72 md:border-r md:border-b-0">
        <header className="flex items-center justify-between gap-3 px-5 pt-4 pb-3 md:pt-5 md:pb-4">
          <h1 className="text-base font-semibold tracking-tight">SignalStack</h1>
          <ConnectionStatus status={status} authState={authState} />
        </header>

        <div className="mx-5 mb-4 hidden items-center gap-3 rounded-xl md:flex border border-line bg-paper px-3 py-2.5">
          {clientId ? <Avatar id={clientId} /> : <span className="size-8 rounded-full bg-hover" />}
          <div className="min-w-0 leading-tight">
            <p className="text-xs text-muted">You are</p>
            <p className="truncate font-mono text-[13px]">{clientId || "…"}</p>
          </div>
        </div>

        <PresenceList
          clients={connectedClients}
          selfId={clientId}
          recipient={recipient}
          onSelect={setPicked}
        />
      </aside>

      <main className="flex min-h-0 flex-1 flex-col">
        <div className="flex items-center gap-3 border-b border-line px-4 py-3 sm:px-6">
          <h2 className="text-sm font-medium">
            {recipient === "all" ? (
              "Everyone"
            ) : (
              <>
                Private with{" "}
                <span title={recipient} className="font-mono text-private">{shortId(recipient)}</span>
              </>
            )}
          </h2>
          {recipient !== "all" && (
            <button
              type="button"
              onClick={() => setPicked("all")}
              className="rounded-md px-2 py-1 text-xs text-muted hover:bg-hover focus-visible:outline-2 focus-visible:outline-signal"
            >
              Back to everyone
            </button>
          )}
        </div>

        <MessageLog messages={messages} selfId={clientId} />

        <Composer
          disabled={!ready}
          target={targetLabel}
          onSend={(text) => sendMessage(text, recipient)}
        />
      </main>
    </div>
  );
}
