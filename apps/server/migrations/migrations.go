// Package migrations 内嵌本目录下的版本化 SQL 迁移并提供执行入口
// （embed + golang-migrate）。本目录是数据库结构的唯一可执行落点：
// 应用启动不隐式执行迁移，由 lexi-loop migrate 命令显式调用
// （docs/backend/structure.md §2）。
package migrations

import (
	"embed"
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog/log"
)

const (
	// DirectionUp 应用全部未执行的迁移。
	DirectionUp = "up"
	// DirectionDown 回退指定步数的迁移（步数见 Migrate 的 steps 参数）。
	DirectionDown = "down"
)

// migrationFS 内嵌本目录全部 *.sql 迁移文件。
//
//go:embed *.sql
var migrationFS embed.FS

// Migrate 对 url 指向的 PostgreSQL 数据库执行本目录下的版本化 SQL 迁移。
//
// direction 为 DirectionUp 时应用全部未执行的迁移；为 DirectionDown 时回退
// steps 步迁移（缺省 1 步，步数按当前版本截断为实际可用数），与
// lexi-loop migrate down 的 CLI 契约一致（docs/backend/structure.md §3）。
// 无可执行的迁移动作视为成功（nil）。
func Migrate(url, direction string, steps ...int) error {
	down := 0 // DirectionDown 的请求回退步数；DirectionUp 无步数语义，恒为 0
	switch direction {
	case DirectionUp:
	case DirectionDown:
		down = 1
		if len(steps) > 0 {
			if steps[0] < 1 {
				return fmt.Errorf("down steps must be a positive integer, got %d", steps[0])
			}
			down = steps[0]
		}
	default:
		return fmt.Errorf("unknown migration direction %q (expected %q or %q)", direction, DirectionUp, DirectionDown)
	}

	src, err := iofs.New(migrationFS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return err
	}
	defer m.Close()

	switch direction {
	case DirectionUp:
		err = m.Up()
	case DirectionDown:
		// down 被实际回退步数覆盖，用于日志。
		down, err = runDown(m, src, down)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Error().
			Err(err).
			Msg("failed to run migrations")
		return err
	}

	entry := log.Info().
		Str("direction", direction)
	if down > 0 {
		entry = entry.Int("steps", down)
	}
	entry.Msg("database migrations applied successfully")

	return nil
}

// runDown 回退 requested 步迁移，返回实际回退步数。请求步数先截断为当前
// 版本下实际可用的迁移数（min(requested, available)，与 D009 同一截断语义）：
// 已在最低版本或请求步数超过可用数时，只回退到最低版本并视为无变化。
// golang-migrate 的 Steps 在这两个边界分别返回 os.ErrNotExist 与
// ErrShortLimit，且后者发生在已回退之后，不适合直接暴露给调用方。
func runDown(m *migrate.Migrate, src source.Driver, requested int) (int, error) {
	cur, _, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, migrate.ErrNoChange
	}
	if err != nil {
		return 0, err
	}

	available := 0
	for available < requested {
		prev, err := src.Prev(cur)
		if errors.Is(err, os.ErrNotExist) {
			// cur 已是第一个迁移版本：仍可回退第一版本身（回退后到达最低版本），
			// 与 Steps 的 readDown 在该边界回退 First() 的行为一致。
			available++
			break
		}
		if err != nil {
			return 0, err
		}
		available++
		cur = prev
	}
	if available == 0 {
		return 0, migrate.ErrNoChange
	}
	return available, m.Steps(-available)
}
