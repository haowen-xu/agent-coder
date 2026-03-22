import { createReadStream, createWriteStream } from 'node:fs'
import { cp, mkdir, readdir, rm } from 'node:fs/promises'
import { dirname, extname, resolve } from 'node:path'
import { pipeline } from 'node:stream/promises'
import { fileURLToPath } from 'node:url'
import { createGzip } from 'node:zlib'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const webuiRoot = resolve(__dirname, '..')
const distDir = resolve(webuiRoot, 'dist')
const staticDir = resolve(webuiRoot, '../internal/app/httpserver/static')

const gzipExts = new Set([
  '.js',
  '.css',
  '.html',
  '.json',
  '.svg',
  '.txt',
  '.xml',
  '.map',
  '.ico',
  '.wasm',
])

async function collectFiles(dir) {
  const entries = await readdir(dir, { withFileTypes: true })
  const files = []

  for (const entry of entries) {
    const fullPath = resolve(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...(await collectFiles(fullPath)))
      continue
    }
    files.push(fullPath)
  }

  return files
}

function shouldGzip(filePath) {
  if (filePath.endsWith('.gz')) {
    return false
  }
  return gzipExts.has(extname(filePath))
}

async function writeGzipFile(filePath) {
  await pipeline(
    createReadStream(filePath),
    createGzip({ level: 9 }),
    createWriteStream(`${filePath}.gz`),
  )
}

async function run() {
  const files = await collectFiles(distDir)
  const candidates = files.filter(shouldGzip)

  await Promise.all(candidates.map((file) => writeGzipFile(file)))

  await rm(staticDir, { recursive: true, force: true })
  await mkdir(staticDir, { recursive: true })
  await cp(distDir, staticDir, { recursive: true })

  console.log(`gzip generated: ${candidates.length} files`) // eslint-disable-line no-console
  console.log(`synced static assets to: ${staticDir}`) // eslint-disable-line no-console
}

run().catch((err) => {
  // eslint-disable-next-line no-console
  console.error(err)
  process.exit(1)
})
