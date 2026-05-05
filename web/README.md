# Content Backend Web

Vite + React + TypeScript 最小前端验收界面。

第一版用于本地浏览器验收后端 API 主链路，不负责生产静态部署，也不复制后端权限、状态流转和一致性规则。

## Requirements

- Node.js 20.19+ 或 22.12+
- npm
- 后端服务可通过 `http://127.0.0.1:8080` 访问

`web/go.mod` 只用于阻止仓库根目录的 `go test ./...` 扫描 `node_modules`，不是前端运行时依赖。

## Development

```bash
npm install
npm run dev -- --host 127.0.0.1
```

浏览器访问：

```text
http://127.0.0.1:5173
```

Vite dev server 会把以下 API 请求代理到后端：

```text
/register
/login
/articles
/me
/healthz
```

## Verification

```bash
npm run lint
npm run build
```

## Current Scope

- 注册
- 登录 / 注销
- 公开文章列表
- 公开文章详情
- 我的文章列表
- 创建草稿
- 编辑草稿
- 发布文章
- 删除文章
