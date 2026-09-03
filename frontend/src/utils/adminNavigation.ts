function normalizePath(path: string): string {
  return path.startsWith('/') ? path : `/${path}`
}

type AdminNavigationQueryValue = string | number | boolean | null | undefined

export type AdminNavigationTarget = string | {
  path: string
  query?: Record<string, AdminNavigationQueryValue>
}

function appendQuery(path: string, query?: Record<string, AdminNavigationQueryValue>): string {
  const entries = Object.entries(query || {})
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
  if (!entries.length) return path
  const params = new URLSearchParams()
  entries.forEach(([key, value]) => params.set(key, String(value)))
  return `${path}${path.includes('?') ? '&' : '?'}${params.toString()}`
}

export function buildAdminURL(target: AdminNavigationTarget): string {
  const rawPath = typeof target === 'string'
    ? target
    : appendQuery(target.path, target.query)
  const [pathnameWithMaybeQuery, hash = ''] = rawPath.split('#')
  const [pathname, search = ''] = pathnameWithMaybeQuery.split('?')
  const normalized = normalizePath(pathname)
  const explicit = (import.meta.env.VITE_ADMIN_URL || import.meta.env.VITE_ADMIN_BASE_URL || '').trim()
  if (explicit) {
    const base = explicit.replace(/\/+$/, '')
    return `${base}${normalized}${search ? `?${search}` : ''}${hash ? `#${hash}` : ''}`
  }

  if (import.meta.env.DEV && typeof window !== 'undefined') {
    const url = new URL(window.location.href)
    if (url.port === '8081') {
      url.port = '8082'
      url.pathname = `/admin${normalized}`
      url.search = search ? `?${search}` : ''
      url.hash = hash ? `#${hash}` : ''
      return url.toString()
    }
  }

  return `/admin${normalized}${search ? `?${search}` : ''}${hash ? `#${hash}` : ''}`
}

export function navigateToAdmin(target: AdminNavigationTarget): void {
  window.location.href = buildAdminURL(target)
}
