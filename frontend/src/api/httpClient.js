// API client backed by the REST backend. Errors carry the server's message and the HTTP `status`.
export function createHttpClient({ baseUrl = '/api' } = {}) {
  async function request(path, { method = 'GET', body, signal } = {}) {
    const response = await fetch(baseUrl + path, {
      method,
      signal,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
    const data = await response.json().catch(() => null)
    if (!response.ok) {
      const error = new Error(data?.error || `Request failed with status ${response.status}`)
      error.status = response.status
      throw error
    }
    return data
  }

  return {
    auth: {
      me: ({ signal } = {}) => request('/auth/me', { signal }),
      login: (email, password) => request('/auth/login', { method: 'POST', body: { email, password } }),
      register: (email, password) => request('/auth/register', { method: 'POST', body: { email, password } }),
      logout: () => request('/auth/logout', { method: 'POST' }),
    },
    jobs: {
      // Newest first; pass the previous page's `next_before` to get the following page.
      list: (before, { signal } = {}) =>
        request(before ? `/jobs?before=${encodeURIComponent(before)}` : '/jobs', { signal }),
      // At most 100 ids per call; resolves to `{ jobs }`, skipping unknown ids.
      listByIds: (ids, { signal } = {}) =>
        request(`/jobs?ids=${ids.map(encodeURIComponent).join(',')}`, { signal }),
      get: (id, { signal } = {}) => request(`/jobs/${encodeURIComponent(id)}`, { signal }),
      create: (job) => request('/jobs', { method: 'POST', body: job }),
      cancel: (id) => request(`/jobs/${encodeURIComponent(id)}/cancel`, { method: 'POST' }),
    },
    admin: {
      listUsers: ({ signal } = {}) => request('/admin/users', { signal }),
      updateUser: (id, changes) => request(`/admin/users/${encodeURIComponent(id)}`, { method: 'PATCH', body: changes }),
    },
  }
}
