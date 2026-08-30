// Seeds a real git remote the running Leakboard container can clone from.
//
// Leakboard only understands GitHub org connections or a plain git clone URL
// (internal/api/repos_handlers.go), so this doesn't call any Leakboard API -
// it just makes an ordinary repo reachable at a file:// path inside the
// leakboard container's own data volume. The demo recording then adds it
// through the real "Add a single repo" UI, same as any other repo.
import { execFileSync } from 'node:child_process'
import { rmSync } from 'node:fs'
import { buildFixtureRepo } from './fixture.mjs'

const SERVICE = process.env.LEAKBOARD_COMPOSE_SERVICE || 'leakboard'
const CONTAINER_PATH = '/home/leakboard/data/seed-repo'

function compose(...args) {
  execFileSync('docker', ['compose', ...args], { cwd: new URL('../..', import.meta.url), stdio: 'inherit' })
}

const localDir = buildFixtureRepo()
console.log(`Built fixture repo at ${localDir}`)

compose('exec', SERVICE, 'rm', '-rf', CONTAINER_PATH)
compose('cp', localDir, `${SERVICE}:${CONTAINER_PATH}`)
compose('exec', '-u', 'root', SERVICE, 'chown', '-R', 'leakboard:leakboard', CONTAINER_PATH)

rmSync(localDir, { recursive: true, force: true })

console.log(`Seeded repo available in-container at file://${CONTAINER_PATH}`)
