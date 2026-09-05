// Package schema 是 bun ORM 映射的持久化结构，仅 repo 子树可导入
// （Go internal 规则约束）。结构只描述数据映射，不含业务逻辑、不定义
// 领域错误；schema ↔ domain 转换只在 repo 内完成
// （docs/backend/structure.md §4.2、docs/specs/backend/Go 单体应用架构规范.md §4.3）。
package schema

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
)

// JSONB 把任意可 JSON 序列化的值 T 映射为 PostgreSQL jsonb 列：
// 写入经 driver.Valuer 序列化为 JSON 文本（PostgreSQL 自动 cast 为 jsonb），
// 读取经 sql.Scanner 反序列化。Val 为零值（nil slice / nil map）时 Value
// 返回 nil（NULL）；NOT NULL DEFAULT 列由 bun 的 nullzero + default tag
// 在插入时省略该列、走服务端默认值。
type JSONB[T any] struct {
	Val T
}

// Value 实现 driver.Valuer：零值映射为 NULL，其余序列化为 JSON 文本。
// 返回 string（而非 []byte），驱动会把它作为文本字面量追加，
// 由 PostgreSQL 解析为 jsonb，避免 bytea 编码歧义。
func (j JSONB[T]) Value() (driver.Value, error) {
	if reflect.ValueOf(j.Val).IsZero() {
		return nil, nil
	}
	b, err := json.Marshal(j.Val)
	if err != nil {
		return nil, fmt.Errorf("schema: marshal jsonb: %w", err)
	}
	return string(b), nil
}

// Scan 实现 sql.Scanner：NULL 保持零值（Val 不动），JSON 文本反序列化进 Val。
func (j *JSONB[T]) Scan(src any) error {
	if src == nil {
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("schema: scan jsonb: unsupported source type %T", src)
	}
	if err := json.Unmarshal(b, &j.Val); err != nil {
		return fmt.Errorf("schema: unmarshal jsonb: %w", err)
	}
	return nil
}
