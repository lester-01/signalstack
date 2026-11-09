"use client";

import { useState } from "react";
import { shortId } from "@/lib/identity";
import { cn } from "@/lib/utils";

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
      className="border-t border-line bg-panel px-4 py-3 sm:px-6"
    >
      <div className="mx-auto flex max-w-2xl items-end gap-2">
        <label htmlFor="composer" className="sr-only">
          Message to {target}
        </label>
        <textarea
          id="composer"
          rows={1}
          value={text}
          maxLength={MAX_LENGTH}
          disabled={disabled}
          placeholder={disabled ? "Waiting for the connection…" : `Message ${target === "everyone" ? target : shortId(target)}`}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              submit();
            }
          }}
          className={cn(
            "max-h-32 min-h-10 flex-1 resize-none rounded-xl border border-line bg-paper px-3.5 py-2 text-[15px]",
            "placeholder:text-muted focus-visible:border-signal focus-visible:outline-2 focus-visible:outline-signal/40",
            "disabled:cursor-not-allowed disabled:opacity-60"
          )}
          style={{ fieldSizing: "content" } as React.CSSProperties}
        />
        <button
          type="submit"
          disabled={disabled || !trimmed}
          className="h-10 rounded-xl bg-signal px-4 text-sm font-medium text-white transition-opacity hover:opacity-90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-signal disabled:cursor-not-allowed disabled:opacity-40"
        >
          Send
        </button>
      </div>
    </form>
  );
}
