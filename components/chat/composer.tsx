"use client";

import { useState } from "react";
import { shortId } from "@/lib/identity";
import { cn } from "@/lib/utils";
import { ArrowUp } from "./icons";

const MAX_LENGTH = 1000; // matches the server's limit

export function Composer({
  disabled,
  target,
  onSend,
}: {
  disabled: boolean;
  target: string;
  onSend: (text: string) => void;
}) {
  const [text, setText] = useState("");
  const trimmed = text.trim();
  const label = target === "all" ? "everyone" : shortId(target);

  const submit = () => {
    if (!trimmed || disabled) return;
    onSend(trimmed);
    setText("");
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
      className="pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-panel via-panel/90 to-transparent px-3 pt-8 pb-[max(0.75rem,env(safe-area-inset-bottom))] sm:px-8"
    >
      <div className="pointer-events-auto mx-auto flex max-w-2xl items-end gap-2 rounded-[1.75rem] bg-bubble p-1.5 shadow-[0_8px_30px_-8px_rgb(0_0_0/0.25)] ring-1 ring-line focus-within:ring-2 focus-within:ring-signal/60">
        <label htmlFor="composer" className="sr-only">
          Message to {label}
        </label>
        <textarea
          id="composer"
          rows={1}
          value={text}
          maxLength={MAX_LENGTH}
          disabled={disabled}
          placeholder={disabled ? "Waiting for the connection…" : `Message ${label}`}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              submit();
            }
          }}
          className={cn(
            "max-h-32 min-h-10 flex-1 resize-none bg-transparent px-3.5 py-2 text-base outline-none sm:text-[15px]",
            "placeholder:text-muted disabled:cursor-not-allowed disabled:opacity-60"
          )}
          style={{ fieldSizing: "content" } as React.CSSProperties}
        />
        <button
          type="submit"
          aria-label="Send message"
          disabled={disabled || !trimmed}
          className="grid size-10 shrink-0 place-items-center rounded-full bg-signal text-white transition-opacity hover:opacity-90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-signal disabled:opacity-30"
        >
          <ArrowUp />
        </button>
      </div>
    </form>
  );
}
