package repo

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// 本项目依赖的 SQLSTATE 错误码（pgx v5 不再导出命名常量，取值见
// PostgreSQL 文档附录 A）。按错误码归一化，不比较驱动错误文本。
const (
	pgUniqueViolation      = "23505" // 唯一约束冲突
	pgForeignKeyViolation  = "23503" // 外键约束冲突
	pgSerializationFailure = "40001" // 序列化失败，整个事务可重试
	pgDeadlockDetected     = "40P01" // 死锁，整个事务可重试
)

// Repo 可识别错误（docs/specs/backend/Go 单体应用架构规范.md §4.5）：
// Service 通过 errors.Is 识别并映射为 apperr.Error；数据库约束类错误按
// SQLSTATE 归一化，未预期的基础设施错误保留原始 cause 向上返回。
var (
	// ErrNotFound：查询无记录（sql.ErrNoRows 或 UPDATE / DELETE 未命中行）。
	ErrNotFound = errors.New("repo: not found")
	// ErrAlreadyExists：唯一约束冲突（如 headword 重复、active session 已存在）。
	ErrAlreadyExists = errors.New("repo: unique constraint violation")
	// ErrReferenceViolation：外键约束冲突（引用了不存在的行）。
	ErrReferenceViolation = errors.New("repo: foreign key violation")
	// ErrConflict：死锁或序列化失败；只有明确可重放的整个事务可重试
	// （docs/backend/structure.md §5）。
	ErrConflict = errors.New("repo: serialization or deadlock conflict")
)

// normalizeError 把一次数据库操作的错误归一化为可识别错误：查无记录、
// 约束类错误按 SQLSTATE 判断，死锁 / 序列化失败映射为 ErrConflict；
// 其余错误原样包装（保留 cause）。op 描述操作，仅用于错误文本与日志。
func normalizeError(op string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("repo: %s: %w", op, ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return fmt.Errorf("repo: %s: %w (%s)", op, ErrAlreadyExists, pgErr.ConstraintName)
		case pgForeignKeyViolation:
			return fmt.Errorf("repo: %s: %w (%s)", op, ErrReferenceViolation, pgErr.ConstraintName)
		case pgSerializationFailure, pgDeadlockDetected:
			return fmt.Errorf("repo: %s: %w", op, ErrConflict)
		}
	}
	return fmt.Errorf("repo: %s: %w", op, err)
}

// escapeLike 转义 LIKE / ILIKE 模式中的通配符（%、_ 与转义符本身），
// 使 search 等用户输入按字面匹配；PostgreSQL 的默认转义符为反斜杠。
func escapeLike(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\', '%', '_':
			out = append(out, '\\', c)
		default:
			out = append(out, c)
		}
	}
	return string(out)
}
