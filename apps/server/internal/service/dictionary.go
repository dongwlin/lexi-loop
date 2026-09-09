package service

import (
	"context"

	"github.com/dongwlin/lexi-loop/apps/server/internal/domain"
)

// Dictionary 提供词典查询用例，不向调用方暴露数据库或事务类型。
type Dictionary interface {
	Lookup(context.Context, string) (*domain.DictionaryEntry, error)
}
