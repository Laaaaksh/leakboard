// Drives a real, running Leakboard instance through the browser and records
// it with Playwright. Requires the stack to already be up and reachable at
// BASE_URL, and the seed repo already placed in-container by seed.mjs.
//
// This is intentionally not a Playwright *test* - it's a scripted walkthrough
// that produces a video, run standalone via `node record.mjs`.
import { chromium } from '@playwright/test'
import { mkdirSync, renameSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const REPO_ROOT = path.resolve(__dirname, '..', '..')

const BASE_URL = process.env.DEMO_BASE_URL || 'http://localhost:8080'
const ADMIN_EMAIL = process.env.DEMO_ADMIN_EMAIL || 'demo@leakboard.dev'
const ADMIN_PASSWORD = process.env.DEMO_ADMIN_PASSWORD || 'DemoWalkthrough123!'
const REPO_NAME = 'payments-service'
const CLONE_URL = 'file:///home/leakboard/data/seed-repo'

const OUT_DIR = path.join(__dirname, 'output')
const VIDEO_DIR = path.join(OUT_DIR, 'video')
mkdirSync(VIDEO_DIR, { recursive: true })

const pause = (page, ms) => page.waitForTimeout(ms)
const type = (locator, text, delay = 45) => locator.pressSequentially(text, { delay })

async function main() {
  const browser = await chromium.launch()
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 2,
    recordVideo: { dir: VIDEO_DIR, size: { width: 1280, height: 800 } },
  })
  const page = await context.newPage()
  const nav = page.locator('nav')

  // 1. First-run setup - creates the single admin account for this instance.
  await page.goto(BASE_URL)
  await page.getByLabel('Email').waitFor()
  await pause(page, 1500)
  await type(page.getByLabel('Email'), ADMIN_EMAIL)
  await type(page.getByLabel('Password'), ADMIN_PASSWORD)
  await pause(page, 700)
  await page.getByRole('button', { name: 'Create account' }).click()

  await page.getByRole('heading', { name: 'Findings' }).waitFor()
  await pause(page, 3800)

  // 2. Track the demo repo (a real local git remote seeded by seed.mjs).
  await nav.getByRole('link', { name: 'Repos' }).click()
  await page.getByRole('heading', { name: 'Add a single repo' }).waitFor()
  await pause(page, 1800)

  const addRepoPanel = page.locator('section.panel', {
    has: page.getByRole('heading', { name: 'Add a single repo' }),
  })
  await type(addRepoPanel.getByPlaceholder('Display name'), REPO_NAME)
  await type(addRepoPanel.getByPlaceholder('Clone URL (https://...)'), CLONE_URL, 20)
  await pause(page, 700)
  await addRepoPanel.getByRole('button', { name: 'Add repo' }).click()

  const repoRow = page.locator('tr', { hasText: REPO_NAME })
  await repoRow.waitFor()
  await pause(page, 2200)

  // 3. Scan it - a real Gitleaks run against the fixture repo's history.
  const [scanResponse] = await Promise.all([
    page.waitForResponse((r) => /\/api\/repos\/\d+\/scan$/.test(r.url()) && r.request().method() === 'POST'),
    repoRow.getByRole('button', { name: 'Scan now' }).click(),
  ])
  if (!scanResponse.ok()) {
    throw new Error(`scan request failed: ${scanResponse.status()} ${await scanResponse.text()}`)
  }
  await pause(page, 3000)

  // 4. Findings land with file, line, and rule attribution.
  await nav.getByRole('link', { name: 'Findings' }).click()
  await page.getByRole('link', { name: 'aws-access-token' }).waitFor()
  await pause(page, 5000)

  // 5. Triage: acknowledge the real leaked AWS key as a true positive.
  await page.getByRole('link', { name: 'aws-access-token' }).click()
  await page.getByRole('heading', { name: 'aws-access-token' }).waitFor()
  await pause(page, 3600)
  await page.getByRole('button', { name: 'Acknowledge' }).click()
  await page.locator('.status-badge', { hasText: 'Acknowledged' }).waitFor()
  await pause(page, 1800)

  // 6. Triage: dismiss the Stripe test-fixture key as a false positive.
  await page.getByRole('link', { name: '← Back to findings' }).click()
  await page.getByRole('link', { name: 'stripe-access-token' }).waitFor()
  await pause(page, 1500)
  await page.getByRole('link', { name: 'stripe-access-token' }).click()
  await page.getByRole('heading', { name: 'stripe-access-token' }).waitFor()
  await pause(page, 3000)
  await page.getByRole('button', { name: 'False positive' }).click()
  await page.locator('.status-badge', { hasText: 'False positive' }).waitFor()
  await pause(page, 2500)

  // 7. Back on the dashboard, the status counts reflect the triage.
  await page.getByRole('link', { name: '← Back to findings' }).click()
  await page.getByRole('heading', { name: 'Findings' }).waitFor()
  await pause(page, 5000)

  // 8. Rule configuration is tunable - add an allowlist rule live.
  await nav.getByRole('link', { name: 'Allowlist' }).click()
  await page.getByRole('heading', { name: 'Add a rule' }).waitFor()
  await pause(page, 1800)
  await type(page.getByPlaceholder('e.g. generic-api-key'), 'stripe-access-token')
  await type(page.getByPlaceholder('e.g. testdata/**'), 'test/fixtures/**')
  await type(page.getByPlaceholder('Why is this safe to ignore?'), 'Test-mode fixtures, never live keys', 25)
  await pause(page, 900)
  await page.getByRole('button', { name: 'Add rule' }).click()
  await page.getByText('stripe-access-token').first().waitFor()
  await pause(page, 4000)

  const video = page.video()
  await context.close()
  await browser.close()

  const recordedPath = video ? await video.path() : null
  if (recordedPath) {
    const dest = path.join(OUT_DIR, 'demo.webm')
    renameSync(recordedPath, dest)
    console.log(`Recording saved to ${dest}`)
  } else {
    throw new Error('no video was recorded')
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
