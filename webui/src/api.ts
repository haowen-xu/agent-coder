export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE'

export async function apiRequest<T>(
  path: string,
  options: {
    method?: HttpMethod
    token?: string
    body?: unknown
  } = {},
): Promise<T> {
  const method = options.method ?? 'GET'
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`
  }

  const res = await fetch(path, {
    method,
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  })

  const text = await res.text()
  const json = text ? JSON.parse(text) : {}
  if (!res.ok) {
    const msg = typeof json?.error === 'string' ? json.error : `HTTP ${res.status}`
    throw new Error(msg)
  }
  return json as T
}
