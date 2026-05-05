import type {
  ArticleDetail,
  ArticleSummary,
  CreateArticleResponse,
  LoginResponse,
  RequestOptions,
} from './types'

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

function messageForStatus(status: number): string {
  switch (status) {
    case 400:
      return '请求内容无法被后端解析，请检查输入。'
    case 401:
      return '需要登录，或当前登录态已经失效。'
    case 403:
      return '当前账号没有执行这个操作的权限。'
    case 404:
      return '资源不存在，可能已经被删除或尚未发布。'
    case 409:
      return '当前状态不允许执行这个操作。'
    case 429:
      return '请求过于频繁，请稍后再试。'
    default:
      return `请求失败，HTTP ${status}。`
  }
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers()

  if (options.body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }
  if (options.token) {
    headers.set('Authorization', `Bearer ${options.token}`)
  }

  const response = await fetch(path, {
    method: options.method ?? 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  })

  if (!response.ok) {
    throw new ApiError(response.status, messageForStatus(response.status))
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

export function register(email: string, password: string): Promise<{ id: number }> {
  return request('/register', {
    method: 'POST',
    body: { email, password },
  })
}

export function login(email: string, password: string): Promise<LoginResponse> {
  return request('/login', {
    method: 'POST',
    body: { email, password },
  })
}

export function listPublishedArticles(): Promise<ArticleSummary[]> {
  return request('/articles')
}

export function getArticle(id: number, token?: string): Promise<ArticleDetail> {
  return request(`/articles/${id}`, { token })
}

export function listMyArticles(token: string): Promise<ArticleSummary[]> {
  return request('/me/articles', { token })
}

export function createArticle(
  token: string,
  title: string,
  content: string,
): Promise<CreateArticleResponse> {
  return request('/articles', {
    token,
    method: 'POST',
    body: { title, content },
  })
}

export function updateArticle(
  token: string,
  id: number,
  title: string,
  content: string,
): Promise<void> {
  return request(`/me/articles/${id}`, {
    token,
    method: 'PUT',
    body: { title, content },
  })
}

export function publishArticle(token: string, id: number): Promise<void> {
  return request('/articles/publish', {
    token,
    method: 'POST',
    body: { article_id: id },
  })
}

export function deleteArticle(token: string, id: number): Promise<void> {
  return request(`/me/articles/${id}`, {
    token,
    method: 'DELETE',
  })
}
