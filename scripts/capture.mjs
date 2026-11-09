#!/usr/bin/env node
// Records the demo assets: screenshots and a side-by-side video of two clients
// chatting. Needs Playwright (npm i -g playwright, or npx playwright install
// chromium) and ffmpeg on PATH.
//
//   1. Start the backend:  cd backend && AUTH_TOKEN=demo go run .
//   2. Start the frontend: npm run dev   (with NEXT_PUBLIC_AUTH_TOKEN=demo)
//   3. node scripts/capture.mjs
//
// Env: APP_URL (default http://localhost:3000), OUT (default ./capture-out)
import { chromium } from "playwright";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const APP_URL = process.env.APP_URL ?? "http://localhost:3000";
const OUT = path.resolve(process.env.OUT ?? "capture-out");
const W = 640, H = 720; // each side; the stacked video is 1280x720
fs.mkdirSync(OUT, { recursive: true });
const tmp = fs.mkdtempSync(path.join(OUT, "video-"));

const browser = await chromium.launch();
const open = async (name, scheme = "light") => {
  const ctx = await browser.newContext({
    viewport: { width: W, height: H },
    colorScheme: scheme,
    deviceScaleFactor: 2,
    recordVideo: { dir: path.join(tmp, name), size: { width: W, height: H } },
  });
  const page = await ctx.newPage();
  await page.goto(APP_URL);
  await page.getByText("Connected", { exact: true }).waitFor();
  return { ctx, page };
};

const alice = await open("alice");
const bob = await open("bob");
const pause = (ms) => alice.page.waitForTimeout(ms);
const say = async (p, text) => {
  await p.locator("#composer").pressSequentially(text, { delay: 45 });
  await pause(250);
  await p.keyboard.press("Enter");
  await pause(900);
};

await pause(1200);
await say(alice.page, "Anyone here? Testing the broadcast path.");
await say(bob.page, "Loud and clear. I can see you in the list.");
// Private message: pick the other client in the presence list.
await alice.page.locator("aside ul button").nth(1).click();
await say(alice.page, "This one is just between us.");
await bob.page.locator("aside ul button").nth(1).click();
await say(bob.page, "Got it. Private messages are amber.");
await pause(1500);

await alice.page.screenshot({ path: path.join(OUT, "chat-alice.png") });
await bob.page.screenshot({ path: path.join(OUT, "chat-bob.png") });

// Closing the contexts is what finalizes the video files.
const videos = {};
for (const [name, c] of Object.entries({ alice, bob })) {
  const v = c.page.video();
  await c.ctx.close();
  videos[name] = await v.path();
}
await browser.close();

execFileSync("ffmpeg", [
  "-y", "-loglevel", "error",
  "-i", videos.alice, "-i", videos.bob,
  "-filter_complex", `[0:v]scale=${W}:${H}[a];[1:v]scale=${W}:${H}[b];[a][b]hstack=inputs=2,fps=30[v]`,
  "-map", "[v]", "-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-pix_fmt", "yuv420p",
  path.join(OUT, "two-clients.mp4"),
]);
fs.rmSync(tmp, { recursive: true, force: true });
console.log("wrote", OUT);
