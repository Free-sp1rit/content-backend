package handler

import (
	"testing"
)

type articleIDTestCase struct {
	name    string
	path    string
	wantID  int64
	wantErr error
}

func runArticleIDParserTests(t *testing.T, parser func(string) (int64, error), tests []articleIDTestCase) {
	t.Helper()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			id, err := parser(tt.path)
			assertErrIs(t, err, tt.wantErr)

			if id != tt.wantID {
				t.Fatalf("got id %d, want %d", id, tt.wantID)
			}
		})
	}
}

func TestParseMyArticleID(t *testing.T) {
	runArticleIDParserTests(t, parseMyArticleID, []articleIDTestCase{
		{
			name:   "valid path",
			path:   "/me/articles/123",
			wantID: 123,
		},
		{
			name:    "invalid prefix",
			path:    "/me/article/123",
			wantErr: ErrInvalidArticleID,
		},
		{
			name:    "path includes query string",
			path:    "/me/articles/123?456",
			wantErr: ErrInvalidArticleID,
		},
	})
}

func TestParseArticleID(t *testing.T) {
	runArticleIDParserTests(t, parseArticleID, []articleIDTestCase{
		{
			name:   "valid path",
			path:   "/articles/123",
			wantID: 123,
		},
		{
			name:    "invalid prefix",
			path:    "/article/123",
			wantErr: ErrInvalidArticleID,
		},
		{
			name:    "path includes query string",
			path:    "/articles/123?456",
			wantErr: ErrInvalidArticleID,
		},
	})
}
