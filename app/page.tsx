"use client";

import { useEffect, useState } from "react";
import { useChat, WebSocketStateEnum } from "@/providers/websocketProvider";
import { Avatar, AvatarStack } from "@/components/chat/avatar";
import { Composer } from "@/components/chat/composer";
import { ConnectionPill } from "@/components/chat/connection-status";
import { Close, Users } from "@/components/chat/icons";
import { MessageLog } from "@/components/chat/message-log";
import { PresenceList } from "@/components/chat/presence-list";
import { shortId } from "@/lib/identity";

export default function Page() {
  const { status, authState, messages, clientId, connectedClients, sendMessage } =
    useChat();
  const [picked, setPicked] = useState("all");
  const [sheetOpen, setSheetOpen] = useState(false);

  // If the person we were messaging goes offline, fall back to everyone.
  const recipient =
    picked === "all" || connectedClients.includes(picked) ? picked : "all";
  const ready =
    status === WebSocketStateEnum.OPEN && authState === "authenticated";

  useEffect(() => {
    if (!sheetOpen) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setSheetOpen(false);
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [sheetOpen]);

  const choose = (id: string) => {
    setPicked(id);
    setSheetOpen(false);
  };

  const people = (
    <PresenceList
      clients={connectedClients}
      selfId={clientId}
      recipient={recipient}
      onSelect={choose}
    />
  );

  return (
    <div className="flex h-dvh gap-3 bg-canvas p-0 md:p-3">
      {/* Desktop: people on the left, on the canvas rather than in a card */}
      <aside className="hidden w-72 shrink-0 flex-col md:flex">
        <div className="px-3 pt-2 pb-5">
          <h1 className="text-xl font-semibold tracking-tight">SignalStack</h1>
          <p className="text-sm text-muted">Realtime chat</p>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto">{people}</div>
        <div className="mt-3 flex items-center gap-3 rounded-2xl bg-panel p-3">
          {clientId ? <Avatar id={clientId} /> : <span className="size-9 rounded-full bg-hover" />}
          <div className="min-w-0 leading-tight">
            <p className="text-xs text-muted">You</p>
            <p className="truncate font-mono text-sm" title={clientId}>
              {clientId ? shortId(clientId) : "…"}
            </p>
          </div>
        </div>
      </aside>

      <main className="relative flex min-w-0 flex-1 flex-col overflow-hidden bg-panel md:rounded-[2rem] md:shadow-[0_1px_0_rgb(0_0_0/0.04),0_20px_50px_-20px_rgb(0_0_0/0.2)]">
        <header className="flex items-center justify-between gap-3 px-4 py-3 sm:px-8 sm:py-4">
          <div className="min-w-0">
            <h2 className="truncate text-base font-semibold tracking-tight">
              {recipient === "all" ? (
                "Everyone"
              ) : (
                <>
                  <span className="text-private">Private</span> with{" "}
                  <span className="font-mono" title={recipient}>
                    {shortId(recipient)}
                  </span>
                </>
              )}
            </h2>
            {recipient !== "all" && (
              <button
                type="button"
                onClick={() => setPicked("all")}
                className="text-xs text-muted underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-signal"
              >
                Back to everyone
              </button>
            )}
          </div>
          <div className="flex items-center gap-3">
            <ConnectionPill status={status} authState={authState} />
            <button
              type="button"
              onClick={() => setSheetOpen(true)}
              aria-label={`${connectedClients.length} online. Choose who to message`}
              className="flex items-center gap-2 rounded-full p-1 focus-visible:outline-2 focus-visible:outline-signal md:hidden"
            >
              <AvatarStack ids={connectedClients} max={3} />
              <Users className="text-muted" />
            </button>
          </div>
        </header>

        <MessageLog messages={messages} selfId={clientId} />

        <Composer
          disabled={!ready}
          target={recipient}
          onSend={(text) => sendMessage(text, recipient)}
        />
      </main>

      {/* Mobile: people open as a sheet from the bottom */}
      {sheetOpen && (
        <div className="fixed inset-0 z-20 md:hidden">
          <button
            type="button"
            aria-label="Close"
            className="absolute inset-0 bg-black/40"
            onClick={() => setSheetOpen(false)}
          />
          <div
            role="dialog"
            aria-modal="true"
            aria-label="Who is online"
            className="absolute inset-x-0 bottom-0 max-h-[75dvh] overflow-y-auto rounded-t-[2rem] bg-panel p-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-2xl"
          >
            <div className="mb-2 flex items-center justify-between px-2">
              <h2 className="text-base font-semibold tracking-tight">Who&apos;s online</h2>
              <button
                type="button"
                onClick={() => setSheetOpen(false)}
                aria-label="Close"
                className="grid size-9 place-items-center rounded-full hover:bg-hover focus-visible:outline-2 focus-visible:outline-signal"
              >
                <Close />
              </button>
            </div>
            {people}
          </div>
        </div>
      )}
    </div>
  );
}
