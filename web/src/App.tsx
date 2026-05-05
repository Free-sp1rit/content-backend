import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import {
  ApiError,
  createArticle,
  deleteArticle,
  getArticle,
  listMyArticles,
  listPublishedArticles,
  login,
  publishArticle,
  register,
  updateArticle,
} from './api'
import type { ArticleDetail, ArticleSummary } from './types'
import './App.css'

type View = 'public' | 'mine' | 'editor'
type AuthMode = 'login' | 'register'
type Notice = { tone: 'success' | 'error' | 'info'; text: string } | null

const tokenStorageKey = 'content_backend_web_token'
const emailStorageKey = 'content_backend_web_email'
const draftContentStorageKey = 'content_backend_web_draft_content'

function App() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenStorageKey) ?? '')
  const [accountEmail, setAccountEmail] = useState(
    () => localStorage.getItem(emailStorageKey) ?? '',
  )
  const [authMode, setAuthMode] = useState<AuthMode>('login')
  const [authEmail, setAuthEmail] = useState(accountEmail)
  const [authPassword, setAuthPassword] = useState('')
  const [view, setView] = useState<View>('public')
  const [notice, setNotice] = useState<Notice>(null)
  const [busy, setBusy] = useState('')

  const [publishedArticles, setPublishedArticles] = useState<ArticleSummary[]>([])
  const [selectedArticle, setSelectedArticle] = useState<ArticleDetail | null>(null)
  const [myArticles, setMyArticles] = useState<ArticleSummary[]>([])
  const [editorArticleID, setEditorArticleID] = useState<number | null>(null)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [draftContents, setDraftContents] = useState<Record<number, string>>(() => {
    const raw = localStorage.getItem(draftContentStorageKey)
    if (!raw) {
      return {}
    }
    try {
      return JSON.parse(raw) as Record<number, string>
    } catch {
      return {}
    }
  })

  const authenticated = token !== ''
  const sortedMyArticles = useMemo(
    () => [...myArticles].sort((a, b) => b.id - a.id),
    [myArticles],
  )

  useEffect(() => {
    let ignore = false

    async function loadInitialPublishedArticles() {
      setBusy('正在刷新公开文章')
      try {
        const articles = await listPublishedArticles()
        if (!ignore) {
          setPublishedArticles(articles)
        }
      } catch (error) {
        if (!ignore) {
          setNotice({ tone: 'error', text: errorMessage(error) })
        }
      } finally {
        if (!ignore) {
          setBusy('')
        }
      }
    }

    void loadInitialPublishedArticles()

    return () => {
      ignore = true
    }
  }, [])

  useEffect(() => {
    localStorage.setItem(draftContentStorageKey, JSON.stringify(draftContents))
  }, [draftContents])

  async function run(label: string, action: () => Promise<void>) {
    setBusy(label)
    setNotice(null)
    try {
      await action()
    } catch (error) {
      setNotice({ tone: 'error', text: errorMessage(error) })
    } finally {
      setBusy('')
    }
  }

  async function refreshPublishedArticles() {
    await run('正在刷新公开文章', async () => {
      const articles = await listPublishedArticles()
      setPublishedArticles(articles)
    })
  }

  async function refreshMyArticles() {
    if (!token) {
      setNotice({ tone: 'info', text: '请先登录，再查看作者侧文章。' })
      return
    }
    await run('正在刷新我的文章', async () => {
      const articles = await listMyArticles(token)
      setMyArticles(articles)
    })
  }

  async function handleAuthSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const email = authEmail.trim()
    const password = authPassword

    if (!email || !password) {
      setNotice({ tone: 'error', text: '请输入 email 和 password。' })
      return
    }

    if (authMode === 'register') {
      await run('正在注册', async () => {
        await register(email, password)
        setAuthMode('login')
        setNotice({ tone: 'success', text: '注册成功，请继续登录。' })
      })
      return
    }

    await run('正在登录', async () => {
      const response = await login(email, password)
      setToken(response.token)
      setAccountEmail(email)
      localStorage.setItem(tokenStorageKey, response.token)
      localStorage.setItem(emailStorageKey, email)
      setAuthPassword('')
      setView('mine')
      setNotice({ tone: 'success', text: '登录成功。' })
      await refreshMyArticlesAfterLogin(response.token)
    })
  }

  async function refreshMyArticlesAfterLogin(nextToken: string) {
    const articles = await listMyArticles(nextToken)
    setMyArticles(articles)
  }

  function logout() {
    setToken('')
    setAccountEmail('')
    setMyArticles([])
    setSelectedArticle(null)
    localStorage.removeItem(tokenStorageKey)
    localStorage.removeItem(emailStorageKey)
    setNotice({ tone: 'info', text: '已注销。' })
    setView('public')
  }

  async function openPublicArticle(id: number) {
    await run('正在打开文章', async () => {
      const article = await getArticle(id, token || undefined)
      setSelectedArticle(article)
      setView('public')
    })
  }

  function startCreate() {
    setEditorArticleID(null)
    setTitle('')
    setContent('')
    setView('editor')
    setNotice(null)
  }

  function startEdit(article: ArticleSummary) {
    setEditorArticleID(article.id)
    setTitle(article.title)
    setContent(draftContents[article.id] ?? '')
    setView('editor')
    if (!draftContents[article.id]) {
      setNotice({
        tone: 'info',
        text: '后端当前没有作者侧草稿详情接口，请在保存前重新填写正文。',
      })
    }
  }

  async function submitArticle(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token) {
      setNotice({ tone: 'error', text: '请先登录。' })
      return
    }

    const nextTitle = title.trim()
    const nextContent = content.trim()
    if (!nextTitle || !nextContent) {
      setNotice({ tone: 'error', text: '标题和正文都不能为空。' })
      return
    }

    if (editorArticleID === null) {
      await run('正在创建草稿', async () => {
        const response = await createArticle(token, nextTitle, nextContent)
        setDraftContents((current) => ({ ...current, [response.id]: nextContent }))
        setNotice({ tone: 'success', text: `草稿已创建，ID ${response.id}。` })
        setTitle('')
        setContent('')
        await refreshMyArticlesAfterMutation()
        setView('mine')
      })
      return
    }

    await run('正在保存草稿', async () => {
      await updateArticle(token, editorArticleID, nextTitle, nextContent)
      setDraftContents((current) => ({ ...current, [editorArticleID]: nextContent }))
      setNotice({ tone: 'success', text: '草稿已保存。' })
      await refreshMyArticlesAfterMutation()
      setView('mine')
    })
  }

  async function refreshMyArticlesAfterMutation() {
    const [mine, published] = await Promise.all([listMyArticles(token), listPublishedArticles()])
    setMyArticles(mine)
    setPublishedArticles(published)
  }

  async function handlePublish(articleID: number) {
    if (!token) {
      return
    }
    await run('正在发布文章', async () => {
      await publishArticle(token, articleID)
      setDraftContents((current) => {
        const next = { ...current }
        delete next[articleID]
        return next
      })
      setNotice({ tone: 'success', text: '文章已发布。' })
      await refreshMyArticlesAfterMutation()
    })
  }

  async function handleDelete(articleID: number) {
    if (!token) {
      return
    }
    await run('正在删除文章', async () => {
      await deleteArticle(token, articleID)
      setDraftContents((current) => {
        const next = { ...current }
        delete next[articleID]
        return next
      })
      if (selectedArticle?.id === articleID) {
        setSelectedArticle(null)
      }
      setNotice({ tone: 'success', text: '文章已删除。' })
      await refreshMyArticlesAfterMutation()
    })
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Content Backend</p>
          <h1>内容发布验收台</h1>
        </div>
        <div className="session">
          {authenticated ? (
            <>
              <span>{accountEmail}</span>
              <button type="button" className="secondary-button" onClick={logout}>
                注销
              </button>
            </>
          ) : (
            <span>未登录</span>
          )}
        </div>
      </header>

      <section className="auth-panel">
        <div className="auth-copy">
          <h2>登录态</h2>
          <p>作者侧操作会自动携带 Bearer token；公开列表和详情允许匿名查看。</p>
        </div>
        <form className="auth-form" onSubmit={handleAuthSubmit}>
          <div className="segmented-control" aria-label="认证模式">
            <button
              type="button"
              className={authMode === 'login' ? 'active' : ''}
              onClick={() => setAuthMode('login')}
            >
              登录
            </button>
            <button
              type="button"
              className={authMode === 'register' ? 'active' : ''}
              onClick={() => setAuthMode('register')}
            >
              注册
            </button>
          </div>
          <label>
            Email
            <input
              value={authEmail}
              onChange={(event) => setAuthEmail(event.target.value)}
              placeholder="author@example.com"
              type="email"
              autoComplete="email"
            />
          </label>
          <label>
            Password
            <input
              value={authPassword}
              onChange={(event) => setAuthPassword(event.target.value)}
              placeholder="secret123"
              type="password"
              autoComplete={authMode === 'login' ? 'current-password' : 'new-password'}
            />
          </label>
          <button type="submit" className="primary-button" disabled={busy !== ''}>
            {authMode === 'login' ? '登录' : '注册'}
          </button>
        </form>
      </section>

      <nav className="tabs" aria-label="主视图">
        <button className={view === 'public' ? 'active' : ''} onClick={() => setView('public')}>
          公开文章
        </button>
        <button
          className={view === 'mine' ? 'active' : ''}
          onClick={() => {
            setView('mine')
            void refreshMyArticles()
          }}
        >
          我的文章
        </button>
        <button className={view === 'editor' ? 'active' : ''} onClick={startCreate}>
          新建文章
        </button>
      </nav>

      {notice && <div className={`notice ${notice.tone}`}>{notice.text}</div>}
      {busy && <div className="loading">{busy}</div>}

      {view === 'public' && (
        <PublicArticlesView
          articles={publishedArticles}
          selectedArticle={selectedArticle}
          onRefresh={() => void refreshPublishedArticles()}
          onOpen={(id) => void openPublicArticle(id)}
        />
      )}

      {view === 'mine' && (
        <MyArticlesView
          authenticated={authenticated}
          articles={sortedMyArticles}
          onRefresh={() => void refreshMyArticles()}
          onCreate={startCreate}
          onEdit={startEdit}
          onPublish={(id) => void handlePublish(id)}
          onDelete={(id) => void handleDelete(id)}
          onOpen={(id) => void openPublicArticle(id)}
        />
      )}

      {view === 'editor' && (
        <EditorView
          authenticated={authenticated}
          editing={editorArticleID !== null}
          title={title}
          content={content}
          onTitleChange={setTitle}
          onContentChange={setContent}
          onSubmit={(event) => void submitArticle(event)}
          onCancel={() => setView('mine')}
        />
      )}
    </main>
  )
}

interface PublicArticlesViewProps {
  articles: ArticleSummary[]
  selectedArticle: ArticleDetail | null
  onRefresh: () => void
  onOpen: (id: number) => void
}

function PublicArticlesView({
  articles,
  selectedArticle,
  onRefresh,
  onOpen,
}: PublicArticlesViewProps) {
  return (
    <section className="workspace two-column">
      <section className="panel">
        <div className="panel-header">
          <div>
            <h2>公开文章</h2>
            <p>{articles.length === 0 ? '当前没有已发布文章。' : `${articles.length} 篇已发布`}</p>
          </div>
          <button type="button" className="secondary-button" onClick={onRefresh}>
            刷新
          </button>
        </div>
        <ArticleList articles={articles} emptyText="没有公开文章。" onOpen={onOpen} />
      </section>

      <section className="panel detail-panel">
        <div className="panel-header">
          <div>
            <h2>文章详情</h2>
            <p>点击公开列表中的文章查看正文。</p>
          </div>
        </div>
        {selectedArticle ? (
          <article className="article-detail">
            <div className="article-meta">
              <span className="state published">published</span>
              <span>ID {selectedArticle.id}</span>
            </div>
            <h3>{selectedArticle.title}</h3>
            <p>{selectedArticle.content}</p>
          </article>
        ) : (
          <EmptyState text="尚未选择文章。" />
        )}
      </section>
    </section>
  )
}

interface MyArticlesViewProps {
  authenticated: boolean
  articles: ArticleSummary[]
  onRefresh: () => void
  onCreate: () => void
  onEdit: (article: ArticleSummary) => void
  onPublish: (id: number) => void
  onDelete: (id: number) => void
  onOpen: (id: number) => void
}

function MyArticlesView({
  authenticated,
  articles,
  onRefresh,
  onCreate,
  onEdit,
  onPublish,
  onDelete,
  onOpen,
}: MyArticlesViewProps) {
  if (!authenticated) {
    return (
      <section className="workspace">
        <section className="panel">
          <EmptyState text="请先登录，再查看作者侧文章。" />
        </section>
      </section>
    )
  }

  return (
    <section className="workspace">
      <section className="panel">
        <div className="panel-header">
          <div>
            <h2>我的文章</h2>
            <p>{articles.length === 0 ? '还没有文章。' : `${articles.length} 篇作者侧文章`}</p>
          </div>
          <div className="button-row">
            <button type="button" className="secondary-button" onClick={onRefresh}>
              刷新
            </button>
            <button type="button" className="primary-button" onClick={onCreate}>
              新建
            </button>
          </div>
        </div>
        {articles.length === 0 ? (
          <EmptyState text="创建第一篇草稿后，它会出现在这里。" />
        ) : (
          <div className="article-table">
            {articles.map((article) => (
              <article className="article-row" key={article.id}>
                <div>
                  <div className="article-meta">
                    <span className={`state ${article.state}`}>{article.state}</span>
                    <span>ID {article.id}</span>
                  </div>
                  <h3>{article.title}</h3>
                </div>
                <div className="row-actions">
                  {article.state === 'draft' ? (
                    <>
                      <button type="button" onClick={() => onEdit(article)}>
                        编辑
                      </button>
                      <button type="button" onClick={() => onPublish(article.id)}>
                        发布
                      </button>
                    </>
                  ) : (
                    <button type="button" onClick={() => onOpen(article.id)}>
                      详情
                    </button>
                  )}
                  <button type="button" className="danger-button" onClick={() => onDelete(article.id)}>
                    删除
                  </button>
                </div>
              </article>
            ))}
          </div>
        )}
      </section>
    </section>
  )
}

interface EditorViewProps {
  authenticated: boolean
  editing: boolean
  title: string
  content: string
  onTitleChange: (value: string) => void
  onContentChange: (value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  onCancel: () => void
}

function EditorView({
  authenticated,
  editing,
  title,
  content,
  onTitleChange,
  onContentChange,
  onSubmit,
  onCancel,
}: EditorViewProps) {
  if (!authenticated) {
    return (
      <section className="workspace">
        <section className="panel">
          <EmptyState text="请先登录，再创建或编辑文章。" />
        </section>
      </section>
    )
  }

  return (
    <section className="workspace">
      <section className="panel">
        <div className="panel-header">
          <div>
            <h2>{editing ? '编辑草稿' : '新建草稿'}</h2>
            <p>已发布文章不能在当前后端规则下继续编辑。</p>
          </div>
        </div>
        <form className="article-form" onSubmit={onSubmit}>
          <label>
            标题
            <input value={title} onChange={(event) => onTitleChange(event.target.value)} />
          </label>
          <label>
            正文
            <textarea
              value={content}
              onChange={(event) => onContentChange(event.target.value)}
              rows={10}
            />
          </label>
          <div className="button-row">
            <button type="submit" className="primary-button">
              {editing ? '保存草稿' : '创建草稿'}
            </button>
            <button type="button" className="secondary-button" onClick={onCancel}>
              返回
            </button>
          </div>
        </form>
      </section>
    </section>
  )
}

interface ArticleListProps {
  articles: ArticleSummary[]
  emptyText: string
  onOpen: (id: number) => void
}

function ArticleList({ articles, emptyText, onOpen }: ArticleListProps) {
  if (articles.length === 0) {
    return <EmptyState text={emptyText} />
  }

  return (
    <div className="article-list">
      {articles.map((article) => (
        <button key={article.id} type="button" className="article-card" onClick={() => onOpen(article.id)}>
          <span className={`state ${article.state}`}>{article.state}</span>
          <strong>{article.title}</strong>
          <span>ID {article.id}</span>
        </button>
      ))}
    </div>
  )
}

function EmptyState({ text }: { text: string }) {
  return <div className="empty-state">{text}</div>
}

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  return '操作失败。'
}

export default App
