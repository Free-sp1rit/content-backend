package handler

import (
	"errors"
	"strconv"
	"strings"
)

var ErrInvalidArticleID = errors.New("invalid article id")

func parseArticleID(path string) (int64, error) {
	const prefix = "/articles/"

	if !strings.HasPrefix(path, prefix) {
		return 0, ErrInvalidArticleID
	}

	idPart := strings.TrimPrefix(path, prefix)
	if idPart == "" {
		return 0, ErrInvalidArticleID
	}

	if strings.Contains(idPart, "/") {
		return 0, ErrInvalidArticleID
	}

	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalidArticleID
	}

	return id, nil
}

func parseMyArticleID(path string) (int64, error) {
	const prefix = "/me/articles/"

	if !strings.HasPrefix(path, prefix) {
		return 0, ErrInvalidArticleID
	}

	idPart := strings.TrimPrefix(path, prefix)
	if idPart == "" {
		return 0, ErrInvalidArticleID
	}

	if strings.Contains(idPart, "/") {
		return 0, ErrInvalidArticleID
	}

	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalidArticleID
	}

	return id, nil
}
