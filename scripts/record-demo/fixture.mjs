// Builds the local git repo used as the scan target for the demo recording.
// Every credential below is a syntactically-valid but non-functional fixture
// value (AWS's own "AKIA...16-chars" shape, a random base64-ish string, and a
// Stripe test-key shape) chosen only so Gitleaks' default rules fire on them.
import { execFileSync } from 'node:child_process'
import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import path from 'node:path'

export function buildFixtureRepo() {
  const dir = mkdtempSync(path.join(tmpdir(), 'leakboard-demo-seed-'))
  mkdirSync(path.join(dir, 'config'), { recursive: true })
  mkdirSync(path.join(dir, 'test', 'fixtures'), { recursive: true })

  writeFileSync(
    path.join(dir, 'README.md'),
    '# payments-service\n\nInternal service config used for the Leakboard demo walkthrough.\n',
  )

  writeFileSync(
    path.join(dir, 'config', 'production.yml'),
    [
      '# NOTE: demo fixture for Leakboard\'s own demo recording - fake, non-functional',
      '# credentials, committed on purpose to show the scanner working end to end.',
      'aws:',
      '  access_key_id: AKIADEMOFAKEKEYZZ234',
      '  secret_access_key: zPq8Rt2Lm9Kx4Vb7Ns3Wc6Yd1Ah5Fg0JmTqLpRxZk',
      '',
    ].join('\n'),
  )

  writeFileSync(
    path.join(dir, 'test', 'fixtures', 'stripe_test_key.rb'),
    [
      '# Test fixture: fake Stripe test-mode key used only in specs.',
      'STRIPE_TEST_KEY = "sk_test_FAKEDEMOKEYNOTREAL0001"',
      '',
    ].join('\n'),
  )

  const git = (...args) => execFileSync('git', args, { cwd: dir, stdio: 'pipe' })
  git('init', '-q')
  git('config', 'user.name', 'Demo Seed')
  git('config', 'user.email', 'demo-seed@example.invalid')
  git('add', '-A')
  git('commit', '-q', '-m', 'Initial commit')

  return dir
}
