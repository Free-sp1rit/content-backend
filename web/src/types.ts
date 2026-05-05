export type ArticleState = 'draft' | 'published'

export interface ArticleSummary {
  id: number
  title: string
  state: ArticleState
  created_at: string
  updated_at: string
}

export interface ArticleDetail extends ArticleSummary {
  author_id: number
  content: string
}

export interface CreateArticleResponse {
  id: number
}

export interface LoginResponse {
  token: string
}

export interface RequestOptions {
  token?: string
  method?: string
  body?: unknown
}
