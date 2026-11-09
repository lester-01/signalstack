# Demo recording plan

This file is the source of truth for the demo video and screenshots. `scripts/capture.mjs` reads the scenario below, so editing the conversation here changes what gets recorded. To redo the demo, run the script; you should not need to describe anything again.

## What gets produced

Each person is recorded on their own, so you can stitch them side by side in a video editor. Alice and Bob run at the same time against the real backend, so the two files line up.

| Output | What it is |
|---|---|
| `desktop/alice.mp4`, `desktop/bob.mp4` | One person each, desktop layout (1280×800), same start time and length |
| `mobile/alice.mp4`, `mobile/bob.mp4` | One person each, phone layout (iPhone 14, 390×844), same start time and length |
| `preview-desktop.mp4`, `preview-mobile.mp4` | Quick side-by-side to check sync. Not for publishing |
| `screenshots/<name>-<person>-<desktop\|mobile>.png` | One per screenshot step, from each person's own window |
| `desktop/manifest.json`, `mobile/manifest.json` | When each step happened (ms from the start), for captions and cuts |

Desktop and mobile are two separate takes, one after the other. Within a take the two people are recorded at the same moment. They are not run in the same chat room, because then the phones would show up in the desktop chat's online list.

## Run it

```bash
# terminal 1: backend, with a token
cd backend && AUTH_TOKEN=demo go run .

# terminal 2: frontend (NEXT_PUBLIC_AUTH_TOKEN must match)
NEXT_PUBLIC_AUTH_TOKEN=demo npm run dev

# terminal 3: record (needs Playwright with Chromium, and ffmpeg)
node scripts/capture.mjs                 # both takes
node scripts/capture.mjs --only=mobile   # one take
```

Environment: `APP_URL` (default `http://localhost:3000`), `OUT` (default `capture-out`).

Restart the backend before recording so the online list starts empty and the timestamps are fresh.

## Recording rules

- Real chat only. Every message goes through the Go server; nothing is faked in the page.
- People type like people. Characters arrive at an uneven pace, there are pauses before they start, some messages get a typo that is fixed with backspace, and one draft is written, deleted and replaced. The `seed` fixes the rhythm, so a rerun looks the same.
- The windows are never touched by anything but the script. No cursor is drawn.
- Light mode for the main takes. Dark mode is a separate pass (see the proposals).
- Video resolution is the viewport size. Playwright cannot record above it, so the phone video is 390×844. Screenshots are taken at 3× on mobile and 2× on desktop and are sharp.

## Scenario

Steps run in order. Only one person acts at a time; the other waits and reads.

| Step | Meaning |
|---|---|
| `wait` | Pause, in ms |
| `say` | Click the composer, type the text, press Enter |
| `fumble` | Used with `say`. Type this misspelled version first, pause, backspace the wrong part, then finish the correct text |
| `draft` | Used with `say`. Type this, pause, delete all of it, then type the real text |
| `pick` | `other` opens a private chat with the other person. `everyone` goes back to the main room. On mobile this opens the people sheet first |
| `shot` | Take a screenshot of every window under this name |

```json
{
  "seed": 7,
  "people": ["alice", "bob"],
  "steps": [
    { "wait": 2500 },
    { "shot": "empty" },
    { "who": "alice", "say": "Morning! Is the build green?", "fumble": "Morning! Is the biuld" },
    { "wait": 1400 },
    { "who": "bob", "say": "Just checked. All green, and the load test finished too.", "draft": "Yes it is, but" },
    { "wait": 1600 },
    { "who": "alice", "say": "Nice. I'll send the staging link in a private chat." },
    { "who": "alice", "pick": "other" },
    { "who": "alice", "say": "Staging is up. Can you open it on your phone?", "fumble": "Staging is up. Can you opne" },
    { "wait": 1500 },
    { "who": "bob", "pick": "other" },
    { "who": "bob", "say": "Opening it now. Looks good on mobile." },
    { "wait": 1200 },
    { "shot": "private" },
    { "who": "bob", "pick": "everyone" },
    { "who": "bob", "say": "Back in the main room. Shipping this afternoon." },
    { "wait": 1200 },
    { "who": "alice", "pick": "everyone" },
    { "who": "alice", "say": "Perfect. Ship it." },
    { "wait": 1800 },
    { "shot": "conversation" },
    { "wait": 1500 }
  ]
}
```

## Screenshots

From the scenario: `empty` (nobody has said anything), `private` (the amber private thread), `conversation` (the finished chat). Each is taken for both people in desktop and mobile.

## Editing notes

- Stitch alice and bob side by side per take. The files start together, so no manual alignment is needed. If you see drift of a few frames, the manifest has the exact step times to line up on.
- Phone clips look best in a device frame at about a third of the canvas height.
- Trim the first and last second if you want a tighter cut. The script leaves half a second of lead-in on purpose.

## Proposed additions (not approved, nothing here is built yet)

Tell me which of these to do.

1. **Typing indicator.** Show "c-4fe961 is typing…" on the other side. It makes the video much livelier, but it needs a new message type on the server and client, so it changes the protocol.
2. **Reconnect clip.** Stop the backend mid-chat, show "Reconnecting", start it again and show the chat recover with the same identity. This demonstrates the resume token, which is the most distinctive thing in the project.
3. **Load test clip.** Run `backend/cmd/loadtest` against the server while a chat is open and show the terminal beside the chat. This needs real hardware to be honest about the numbers.
4. **Live metrics pane.** A small page that polls `/metrics` and shows connections, messages per second and memory during the load test. It needs a tiny proxy for the cross-origin request.
5. **Dark mode pass.** The same scenario in dark mode, for a README that follows the viewer's theme.
6. **README GIFs.** Short loops cut from the video, since GitHub plays GIFs inline but not long videos.
7. **Social preview image.** A 1280×640 image for the repository's link preview.
8. **Visible cursor.** Draw a pointer in the recording so clicks on the people list are easy to follow.
9. **Parallel desktop and mobile takes.** Run both takes at once on two backend instances, so a desktop and a phone clip line up for a combined layout.
