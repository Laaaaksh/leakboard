export interface SessionInfo {
  authenticated: boolean
  setupRequired?: boolean
  email?: string
}

export interface Finding {
  id: number
  repoId: number
  repoName: string
  fingerprint: string
  ruleId: string
  description: string
  filePath: string
  startLine: number
  endLine: number
  commitSha: string
  commitAuthor: string
  commitEmail: string
  secret: string
  status: 'new' | 'acknowledged' | 'resolved' | 'false_positive'
  firstSeenAt: string
  lastSeenAt: string
}

export interface Repo {
  id: number
  name: string
  cloneUrl: string
  defaultBranch: string
  lastScannedAt: string | null
  lastScanError: string
  scanIntervalSecs: number
}

export interface Connection {
  id: number
  name: string
  githubOrg: string
  createdAt: string
}

export interface AllowlistEntry {
  id: number
  ruleId: string
  pathPattern: string
  regex: string
  fingerprint: string
  reason: string
  createdAt: string
}

export interface Webhook {
  id: number
  kind: 'slack' | 'discord' | 'generic'
  targetUrl: string
  enabled: boolean
  createdAt: string
}

export interface ScanRun {
  ID: number
  RepoID: number
  StartedAt: string
  FinishedAt: string | null
  Status: string
  NewFindings: number
  Error: string
}

export interface Stats {
  findingCounts: Partial<Record<Finding['status'], number>>
  repoCount: number
}

class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!res.ok) {
    let message = res.statusText
    try {
      const body = await res.json()
      if (body?.error) message = body.error
    } catch {
      // response had no JSON body; fall back to statusText
    }
    throw new ApiError(res.status, message)
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') {
    return undefined as T
  }
  return (await res.json()) as T
}

export const api = {
  session: () => request<SessionInfo>('/api/session'),
  setup: (email: string, password: string) =>
    request('/api/setup', { method: 'POST', body: JSON.stringify({ email, password }) }),
  login: (email: string, password: string) =>
    request('/api/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
  logout: () => request('/api/logout', { method: 'POST' }),

  stats: () => request<Stats>('/api/stats'),
  scanRuns: () => request<ScanRun[]>('/api/scans'),

  findings: (filter: { repoId?: number; status?: string; ruleId?: string } = {}) => {
    const params = new URLSearchParams()
    if (filter.repoId) params.set('repo_id', String(filter.repoId))
    if (filter.status) params.set('status', filter.status)
    if (filter.ruleId) params.set('rule_id', filter.ruleId)
    const qs = params.toString()
    return request<Finding[]>(`/api/findings${qs ? `?${qs}` : ''}`)
  },
  finding: (id: number, reveal = false) => request<Finding>(`/api/findings/${id}${reveal ? '?reveal=1' : ''}`),
  updateFindingStatus: (id: number, status: Finding['status']) =>
    request(`/api/findings/${id}/status`, { method: 'POST', body: JSON.stringify({ status }) }),

  repos: () => request<Repo[]>('/api/repos'),
  createRepo: (name: string, cloneUrl: string, defaultBranch?: string) =>
    request<Repo>('/api/repos', { method: 'POST', body: JSON.stringify({ name, cloneUrl, defaultBranch }) }),
  deleteRepo: (id: number) => request(`/api/repos/${id}`, { method: 'DELETE' }),
  scanRepoNow: (id: number) => request<{ newFindings: number }>(`/api/repos/${id}/scan`, { method: 'POST' }),

  connections: () => request<Connection[]>('/api/connections'),
  createConnection: (name: string, githubOrg: string, accessToken: string) =>
    request<{ connection: Connection; reposAdded: number }>('/api/connections', {
      method: 'POST',
      body: JSON.stringify({ name, githubOrg, accessToken }),
    }),
  deleteConnection: (id: number) => request(`/api/connections/${id}`, { method: 'DELETE' }),
  syncConnection: (id: number) => request<{ reposAdded: number }>(`/api/connections/${id}/sync`, { method: 'POST' }),

  allowlist: () => request<AllowlistEntry[]>('/api/allowlist'),
  createAllowlistEntry: (entry: { ruleId?: string; pathPattern?: string; regex?: string; reason?: string }) =>
    request<AllowlistEntry>('/api/allowlist', { method: 'POST', body: JSON.stringify(entry) }),
  deleteAllowlistEntry: (id: number) => request(`/api/allowlist/${id}`, { method: 'DELETE' }),

  webhooks: () => request<Webhook[]>('/api/webhooks'),
  createWebhook: (kind: Webhook['kind'], targetUrl: string) =>
    request<Webhook>('/api/webhooks', { method: 'POST', body: JSON.stringify({ kind, targetUrl }) }),
  deleteWebhook: (id: number) => request(`/api/webhooks/${id}`, { method: 'DELETE' }),
}

export { ApiError }
