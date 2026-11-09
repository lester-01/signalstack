#!/usr/bin/env node
// Records the demo from the scenario in docs/DEMO.md: one video per person and
// per layout (desktop, mobile), kept in sync, plus screenshots.
// See docs/DEMO.md for what it produces and how to run it.
// Needs Playwright with Chromium, and ffmpeg on PATH.
import { chromium, devices } from "playwright";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const APP_URL = process.env.APP_URL ?? "http://localhost:3000";
const OUT = path.resolve(process.env.OUT ?? "capture-out");
const LEAD_S = 0.5; // lead-in kept before the first step
const only = process.argv.find((a) => a.startsWith("--only="))?.split("=")[1];

// ---- scenario: the first json block under "## Scenario" in docs/DEMO.md ----
const md = fs.readFileSync(
  path.join(path.dirname(fileURLToPath(import.meta.url)), "../docs/DEMO.md"),
  "utf8"
);
const block = md.split("## Scenario")[1]?.match(/```json\n([\s\S]*?)\n```/);
if (!block) throw new Error("no json block under '## Scenario' in docs/DEMO.md");
const scenario = JSON.parse(block[1]);

// ---- layouts ----
const phone = { ...devices["iPhone 14"] };
delete phone.defaultBrowserType;
const LAYOUTS = {
  desktop: { viewport: { width: 1280, height: 800 }, deviceScaleFactor: 2 },
  mobile: { ...phone, deviceScaleFactor: 3 },
};

// ---- human typing, seeded so reruns have the same rhythm ----
let seed = scenario.seed ?? 1;
const rand = () => {
  seed = (seed + 0x6d2b79f5) | 0;
  let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
  t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
};
const between = (a, b) => a + rand() * (b - a);

async function typeText(page, text) {
  for (const ch of text) {
    await page.keyboard.type(ch);
    const slow = ".,!?".includes(ch) ? between(120, 320) : ch === " " ? between(20, 90) : 0;
    await page.waitForTimeout(between(55, 135) + slow);
  }
}
async function backspace(page, n, fast = false) {
  for (let i = 0; i < n; i++) {
    await page.keyboard.press("Backspace");
    await page.waitForTimeout(fast ? between(35, 60) : between(70, 120));
  }
}

async function say(page, step) {
  const composer = page.locator("#composer");
  await page.waitForTimeout(between(500, 900)); // read, then reach for the keyboard
  await composer.click();
  if (step.draft) {
    await typeText(page, step.draft);
    await page.waitForTimeout(between(800, 1100));
    await backspace(page, step.draft.length, true);
    await page.waitForTimeout(between(500, 800));
  }
  if (step.fumble) {
    let p = 0;
    while (p < step.fumble.length && step.fumble[p] === step.say[p]) p++;
    await typeText(page, step.fumble);
    await page.waitForTimeout(between(450, 750)); // notice the mistake
    await backspace(page, step.fumble.length - p);
    await typeText(page, step.say.slice(p));
  } else {
    await typeText(page, step.say);
  }
  await page.waitForTimeout(between(250, 450));
  await page.keyboard.press("Enter");
  await page.waitForTimeout(between(700, 1000));
}

async function pick(page, which) {
  // On mobile the people list lives in a sheet behind this button.
  const sheetButton = page.getByRole("button", { name: /online/ });
  if (await sheetButton.isVisible()) await sheetButton.click();
  const buttons = page.locator("button[aria-pressed]:visible");
  await buttons.nth(which === "everyone" ? 0 : 1).waitFor({ timeout: 8000 });
  await page.waitForTimeout(between(300, 600));
  await buttons.nth(which === "everyone" ? 0 : 1).click();
  await page.waitForTimeout(between(500, 800));
}

// ---- one take: all people at once, one layout ----
async function take(layout) {
  const dir = path.join(OUT, layout);
  const tmp = path.join(OUT, `.tmp-${layout}`);
  fs.mkdirSync(path.join(OUT, "screenshots"), { recursive: true });
  fs.mkdirSync(dir, { recursive: true });

  const browser = await chromium.launch();
  const people = {};
  await Promise.all(
    scenario.people.map(async (name) => {
      const opts = LAYOUTS[layout];
      const ctx = await browser.newContext({
        ...opts,
        colorScheme: "light",
        recordVideo: { dir: path.join(tmp, name), size: opts.viewport },
      });
      const page = await ctx.newPage();
      const created = Date.now();
      await page.goto(APP_URL);
      await page.getByRole("status").getByText("Live").waitFor({ timeout: 15000 });
      people[name] = { ctx, page, created };
    })
  );

  const start = Date.now();
  const log = [];
  const stamp = (what) => log.push({ t: Date.now() - start, what });

  for (const step of scenario.steps) {
    if (step.wait) {
      await people[scenario.people[0]].page.waitForTimeout(step.wait);
    } else if (step.shot) {
      stamp(`screenshot ${step.shot}`);
      await Promise.all(
        Object.entries(people).map(([name, p]) =>
          p.page.screenshot({
            path: path.join(OUT, "screenshots", `${step.shot}-${name}-${layout}.png`),
          })
        )
      );
    } else if (step.say) {
      stamp(`${step.who} says: ${step.say}`);
      await say(people[step.who].page, step);
      stamp(`${step.who} sent`);
    } else if (step.pick) {
      stamp(`${step.who} picks ${step.pick}`);
      await pick(people[step.who].page, step.pick);
    }
  }
  const end = Date.now();

  const videos = {};
  for (const [name, p] of Object.entries(people)) {
    const v = p.page.video();
    await p.ctx.close(); // finalizes the video file
    videos[name] = { file: await v.path(), created: p.created };
  }
  await browser.close();

  // Trim every video to the same wall-clock window, so they line up.
  const duration = (end - start) / 1000 + LEAD_S;
  for (const [name, v] of Object.entries(videos)) {
    const ss = Math.max(0, (start - v.created) / 1000 - LEAD_S);
    execFileSync("ffmpeg", [
      "-y", "-loglevel", "error", "-ss", ss.toFixed(3), "-i", v.file,
      "-t", duration.toFixed(3), "-r", "30",
      "-c:v", "libx264", "-preset", "veryfast", "-crf", "20", "-pix_fmt", "yuv420p", "-an",
      path.join(dir, `${name}.mp4`),
    ]);
  }
  const names = scenario.people;
  execFileSync("ffmpeg", [
    "-y", "-loglevel", "error",
    ...names.flatMap((n) => ["-i", path.join(dir, `${n}.mp4`)]),
    "-filter_complex",
    names.map((_, i) => `[${i}:v]scale=-2:720[v${i}]`).join(";") +
      `;${names.map((_, i) => `[v${i}]`).join("")}hstack=inputs=${names.length}[o]`,
    "-map", "[o]", "-c:v", "libx264", "-preset", "veryfast", "-crf", "26", "-pix_fmt", "yuv420p",
    path.join(OUT, `preview-${layout}.mp4`),
  ]);
  fs.rmSync(tmp, { recursive: true, force: true });
  fs.writeFileSync(
    path.join(OUT, layout, "manifest.json"),
    JSON.stringify({ layout, durationSeconds: duration, leadSeconds: LEAD_S, events: log }, null, 2)
  );
  console.log(`${layout}: ${names.join(" + ")} recorded, ${duration.toFixed(1)}s each`);
}

fs.mkdirSync(OUT, { recursive: true });
for (const layout of Object.keys(LAYOUTS)) {
  if (!only || only === layout) await take(layout);
}
console.log("wrote", OUT);
