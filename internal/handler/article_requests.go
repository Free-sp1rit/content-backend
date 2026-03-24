package handler

type CreateArticleRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type PublishArticleRequest struct {
	ArticleID int64 `json:"article_id"`
}

type UpdateArticleRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
