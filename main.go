package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
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

func main() {
	mux := http.NewServeMux()

	// API 路由
	mux.HandleFunc("GET /api/health", healthCheck)
	mux.HandleFunc("GET /api/info", serverInfo)
	mux.HandleFunc("POST /api/echo", echo)

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
