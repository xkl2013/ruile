function normalizePath(path: string): string {
  return path.startsWith('/') ? path : `/${path}`
}

export function buildMainAppURL(path: string): string {
  const normalized = normalizePath(path)
  const explicit = (import.meta.env.VITE_ADMIN_MAIN_APP_URL || '').trim()
  if (explicit) {
    const base = explicit.replace(/\/+$/, '')
    return `${base}${normalized}`
  }

  if (import.meta.env.DEV && typeof window !== 'undefined') {
    const url = new URL(window.location.href)
    if (url.port === '8082') {
      url.port = '8081'
      url.pathname = normalized
      url.search = ''
      url.hash = ''
      return url.toString()
    }
  }

  return normalized
}

export function openMainAppPath(path: string): void {
  window.location.href = buildMainAppURL(path)
}
