package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // 注册 pgx 驱动
)

// PgStore 基于 PostgreSQL（Supabase）的存储实现
type PgStore struct {
	db *sql.DB
}

func NewPgStore(dsn string) (*PgStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	// 免费实例资源有限，控制连接数
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// 启动时自动建表，免去手动操作
	const schema = `
		CREATE TABLE IF NOT EXISTS notes (
			id         TEXT PRIMARY KEY,
			content    TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("建表失败: %w", err)
	}
	return &PgStore{db: db}, nil
}

// List 返回全部笔记（按创建时间倒序）
func (s *PgStore) List() ([]Note, error) {
	rows, err := s.db.Query(`SELECT id, content, created_at FROM notes ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Content, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// Add 新增一条笔记
func (s *PgStore) Add(content string) (Note, error) {
	n := Note{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Content:   content,
		CreatedAt: time.Now(),
	}
	_, err := s.db.Exec(
		`INSERT INTO notes (id, content, created_at) VALUES ($1, $2, $3)`,
		n.ID, n.Content, n.CreatedAt,
	)
	return n, err
}

// Delete 按 ID 删除，返回是否存在
func (s *PgStore) Delete(id string) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM notes WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	count, err := res.RowsAffected()
	return count > 0, err
}

func (s *PgStore) Close() error { return s.db.Close() }
