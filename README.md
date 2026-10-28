# rendertest

前端 HTML + Go 后端的单服务架构，可一键部署到 Render。

## 架构说明

采用 **单个 Go Web 服务同时托管前端静态页面和后端 API** 的方式：

- 无需跨域配置（前端与 API 同源）
- Render 上只需创建一个服务，构建快、免费额度友好
- 前端文件在项目根目录，由 Go 的静态文件中间件直接托管

```
rendertest/
├── index.html      # 前端页面（调用 /api/* 接口）
├── main.go         # Go 后端（静态托管 + API）
├── go.mod          # Go 模块定义
└── render.yaml     # Render Blueprint 部署配置
```

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/api/health` | 健康检查（Render Health Check Path） |
| GET  | `/api/info`   | 服务端环境信息 |
| POST | `/api/echo`   | 回显测试，body: `{"message": "xxx"}` |
| GET  | `/api/notes`  | 列出实验数据 |
| POST | `/api/notes`  | 新增，body: `{"content": "xxx"}` |
| DELETE | `/api/notes/{id}` | 删除一条 |

## 数据存储说明

支持两种存储后端，通过环境变量 `DATABASE_URL` 自动切换：

- **配置了 `DATABASE_URL`** → PostgreSQL（Supabase），启动时自动建表，数据持久保存
- **未配置** → 本地 JSON 文件（`data/notes.json`），适合无数据库的快速实验

> 本地开发：在项目根目录创建 `.env` 文件写入 `DATABASE_URL=...`（已 gitignore）。
> Render 部署：在服务设置的 Environment 中添加 `DATABASE_URL`。
> Supabase 连接串请使用 **Session/Transaction Pooler** 地址（`*.pooler.supabase.com`），直连地址（`db.xxx.supabase.co`）仅支持 IPv6。

## 本地运行

```bash
go run .
# 浏览器访问 http://localhost:8080
```

## 部署到 Render

方式一（推荐，Blueprint 自动部署）：
1. 将代码推送到 GitHub/GitLab 仓库
2. Render 控制台 → New + → Blueprint → 选择仓库
3. Render 会读取 `render.yaml` 自动创建服务

方式二（手动创建）：
1. Render 控制台 → New + → Web Service → 选择仓库
2. Runtime 选 `Go`
3. Build Command: `go build -o app .`
4. Start Command: `./app`
5. Health Check Path: `/api/health`

> Render 会自动注入 `PORT` 环境变量，服务已做适配，无需额外配置。
