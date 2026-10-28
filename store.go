package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Note 一条实验数据
type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// FileStore 基于单个 JSON 文件的极简存储，写入时全量重写。
// 仅适合实验数据；并发安全由互斥锁保证。
type FileStore struct {
	mu   sync.Mutex
	path string
}

func NewFileStore(path string) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &FileStore{path: path}, nil
}

func (s *FileStore) load() ([]Note, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []Note{}, nil // 文件不存在视为空数据
	}
	if err != nil {
		return nil, err
	}
	var notes []Note
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

func (s *FileStore) save(notes []Note) error {
	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

// List 返回全部笔记（按创建时间倒序）
func (s *FileStore) List() ([]Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load()
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(notes)-1; i < j; i, j = i+1, j-1 {
		notes[i], notes[j] = notes[j], notes[i]
	}
	return notes, nil
}

// Add 新增一条笔记
func (s *FileStore) Add(content string) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load()
	if err != nil {
		return Note{}, err
	}
	note := Note{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Content:   content,
		CreatedAt: time.Now(),
	}
	notes = append(notes, note)
	return note, s.save(notes)
}

// Delete 按 ID 删除，返回是否存在
func (s *FileStore) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load()
	if err != nil {
		return false, err
	}
	out := notes[:0]
	found := false
	for _, n := range notes {
		if n.ID == id {
			found = true
			continue
		}
		out = append(out, n)
	}
	if !found {
		return false, nil
	}
	return true, s.save(out)
}
