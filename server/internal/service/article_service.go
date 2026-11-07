package service

import (
	"context"

	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/store"
)

type ArticleService struct {
	store *store.Store
}

func NewArticleService(store *store.Store) *ArticleService {
	return &ArticleService{store: store}
}

func (s *ArticleService) List(
	ctx context.Context,
	filter store.ArticleFilter,
) ([]domain.Article, int, error) {
	articles, err := s.store.ListArticles(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.store.CountArticlesFilter(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return articles, total, nil
}

func (s *ArticleService) Get(ctx context.Context, id string) (domain.Article, error) {
	return s.store.GetArticle(ctx, id)
}

func (s *ArticleService) Accounts(ctx context.Context) ([]string, error) {
	return s.store.ListArticleAccounts(ctx)
}
