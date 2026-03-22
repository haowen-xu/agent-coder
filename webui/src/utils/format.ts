function pad2(value: number): string {
  if (value < 10) {
    return `0${value}`
  }
  return String(value)
}

function parseDateTime(input: string | number | Date): Date | null {
  if (input instanceof Date) {
    return Number.isNaN(input.getTime()) ? null : input
  }
  const raw = typeof input === 'string' ? input.trim() : input
  if (raw === '') {
    return null
  }

  let parsed = new Date(raw)
  if (!Number.isNaN(parsed.getTime())) {
    return parsed
  }

  if (typeof raw === 'string' && raw.includes(' ')) {
    parsed = new Date(raw.replace(' ', 'T'))
    if (!Number.isNaN(parsed.getTime())) {
      return parsed
    }
  }
  return null
}

export function formatLocalDateTime(
  value: string | number | Date | null | undefined,
  fallback = '-',
): string {
  if (value === null || value === undefined) {
    return fallback
  }
  const date = parseDateTime(value)
  if (!date) {
    return fallback
  }
  const year = date.getFullYear()
  const month = pad2(date.getMonth() + 1)
  const day = pad2(date.getDate())
  const hour = pad2(date.getHours())
  const minute = pad2(date.getMinutes())
  const second = pad2(date.getSeconds())
  return `${year}-${month}-${day} ${hour}:${minute}:${second}`
}
