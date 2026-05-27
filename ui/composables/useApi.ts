// Auth-aware HTTP client. Wraps `$fetch` with:
//   - Bearer JWT (from the Pinia auth store) injected
//   - JSON error normalisation
//   - Automatic redirect to /login on 401
//
// Use `api.get(path, options)`, `api.post(path, body, options)`, etc.
// For OData queries, pass `query` as an object — keys starting with `$` are
// forwarded verbatim ($filter, $top, $orderby, etc.).

import { useAuthStore } from '~/stores/auth'

type Method = 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'

interface RequestOptions {
  body?: unknown
  query?: Record<string, unknown>
  headers?: Record<string, string>
  responseType?: 'json' | 'blob' | 'arrayBuffer' | 'text'
  signal?: AbortSignal
}

export function useApi() {
  const cfg = useRuntimeConfig()
  const auth = useAuthStore()
  const base = cfg.public.apiBase as string

  async function call<T = unknown>(method: Method, path: string, opts: RequestOptions = {}): Promise<T> {
    const headers: Record<string, string> = {
      ...(opts.headers ?? {}),
    }
    if (auth.token) headers['Authorization'] = `Bearer ${auth.token}`
    if (opts.body !== undefined && !(opts.body instanceof FormData)) {
      headers['content-type'] ??= 'application/json'
    }

    try {
      return await $fetch<T>(path, {
        baseURL: base,
        method,
        body: opts.body as any,
        query: opts.query,
        headers,
        responseType: opts.responseType as any,
        signal: opts.signal,
      })
    } catch (err: any) {
      const status = err?.response?.status ?? err?.statusCode
      if (status === 401 && typeof window !== 'undefined') {
        auth.logout()
        await navigateTo('/login')
      }
      throw err
    }
  }

  return {
    get:    <T = unknown>(path: string, opts?: RequestOptions) => call<T>('GET', path, opts),
    post:   <T = unknown>(path: string, body?: unknown, opts?: RequestOptions) => call<T>('POST', path, { ...opts, body }),
    patch:  <T = unknown>(path: string, body?: unknown, opts?: RequestOptions) => call<T>('PATCH', path, { ...opts, body }),
    put:    <T = unknown>(path: string, body?: unknown, opts?: RequestOptions) => call<T>('PUT', path, { ...opts, body }),
    delete: <T = unknown>(path: string, opts?: RequestOptions) => call<T>('DELETE', path, opts),
  }
}

/** OData list response shape returned by /odata/:set */
export interface ODataList<T = Record<string, any>> {
  value: T[]
  '@odata.count'?: number
}

/**
 * Convenience composable: useEntity('templates') returns CRUD helpers bound to
 * the /odata/templates collection.
 */
export function useEntity<T = Record<string, any>>(set: string) {
  const api = useApi()
  const path = `/odata/${set}`
  return {
    list: (q: Record<string, unknown> = {}) => api.get<ODataList<T>>(path, { query: q }),
    get:  (shortid: string) => api.get<T>(`${path}/${shortid}`),
    create: (body: Partial<T>) => api.post<T>(path, body),
    update: (shortid: string, body: Partial<T>) => api.patch<T>(`${path}/${shortid}`, body),
    remove: (shortid: string) => api.delete<void>(`${path}/${shortid}`),
  }
}
