import { readFile, unlink } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

interface RuntimeState {
  serverPid: number
  workerPid: number
}

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const STATE_PATH = path.resolve(__dirname, '../.playwright-e2e-state.json')

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function tryKill(pid: number, signal: NodeJS.Signals): boolean {
  if (!Number.isInteger(pid) || pid <= 0) {
    return false
  }

  try {
    process.kill(-pid, signal)
    return true
  } catch {
    // fallback to single pid below
  }

  try {
    process.kill(pid, signal)
    return true
  } catch {
    return false
  }
}

async function stopProcess(pid: number) {
  const sent = tryKill(pid, 'SIGTERM')
  if (!sent) {
    return
  }

  await sleep(1_000)
  tryKill(pid, 'SIGKILL')
}

export default async function globalTeardown() {
  let state: RuntimeState | null = null

  try {
    const text = await readFile(STATE_PATH, 'utf-8')
    state = JSON.parse(text) as RuntimeState
  } catch {
    state = null
  }

  if (state) {
    await stopProcess(state.workerPid)
    await stopProcess(state.serverPid)
  }

  try {
    await unlink(STATE_PATH)
  } catch {
    // ignore
  }
}
