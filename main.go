package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var startTime = time.Now()

// writeJSON 统一的 JSON 响应输出
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// healthCheck 健康检查接口，Render 可将其配置为 Health Check Path
func healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "rendertest-go",
		"time":    time.Now().Format(time.RFC3339),
		"uptime":  time.Since(startTime).Round(time.Second).String(),
	})
}

// serverInfo 返回服务端环境信息，用于验证部署环境
func serverInfo(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]any{
		"hostname":   hostname,
		"goVersion":  "go1.26",
		"port":       os.Getenv("PORT"),
		"renderEnv":  os.Getenv("RENDER"), // Render 平台会注入该环境变量
		"clientAddr": r.RemoteAddr,
	})
}

// echo 回显接口，演示前端 POST 请求与 JSON 解析
func echo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "仅支持 POST 请求"})
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无效的 JSON 请求体"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"echo":    req.Message,
		"from":    "Go 后端服务",
		"at":      time.Now().Format("2006-01-02 15:04:05"),
	})
}

// notesList GET /api/notes — 列出全部笔记
func (s *app) notesList(w http.ResponseWriter, r *http.Request) {
	notes, err := s.store.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

// notesAdd POST /api/notes — 新增笔记
func (s *app) notesAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content 不能为空"})
		return
	}
	note, err := s.store.Add(req.Content)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

// notesDelete DELETE /api/notes/{id} — 删除笔记
func (s *app) notesDelete(w http.ResponseWriter, r *http.Request) {
	found, err := s.store.Delete(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "笔记不存在"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type app struct {
	store *PgStore
}

// loadDotEnv 极简 .env 加载器，本地开发时从 .env 文件读取环境变量
func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return // 文件不存在就跳过（Render 上靠控制台配置环境变量）
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

func main() {
	loadDotEnv(".env")

	// 连接数据库（连接串从环境变量读取，不写死在代码里）
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("未配置 DATABASE_URL 环境变量，无法连接数据库")
	}
	pg, err := NewPgStore(dsn)
	if err != nil {
		log.Fatalf("连接 PostgreSQL 失败: %v", err)
	}
	defer pg.Close()
	a := &app{store: pg}

	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("GET /api/health", healthCheck)
	mux.HandleFunc("GET /api/info", serverInfo)
	mux.HandleFunc("POST /api/echo", echo)
	mux.HandleFunc("GET /api/notes", a.notesList)
	mux.HandleFunc("POST /api/notes", a.notesAdd)
	mux.HandleFunc("DELETE /api/notes/{id}", a.notesDelete)

	// 静态文件：托管前端 HTML（默认返回 index.html）
	mux.Handle("/", http.FileServer(http.Dir(".")))

	// Render 平台通过 PORT 环境变量指定监听端口，本地默认 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("rendertest-go 服务已启动，监听端口 :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
