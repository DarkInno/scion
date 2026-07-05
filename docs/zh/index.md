---
layout: home

hero:
  name: Scion
  text: 复制粘贴 Go 后端模块
  tagline: 显式依赖、安全优先、AI友好。复制生产就绪的代码到你的项目中。
  actions:
    - theme: brand
      text: 快速开始
      link: /zh/guide/getting-started
    - theme: alt
      text: 查看模块
      link: /zh/modules/
    - theme: alt
      text: GitHub
      link: https://github.com/DarkInno/scion

features:
  - icon: 🔐
    title: Auth 认证
    details: JWT 认证 + bcrypt，限流，用户枚举防护。
    link: /zh/modules/auth
  - icon: 📦
    title: CRUD 增删改查
    details: 通用 CRUD 操作 + 分页，排序/过滤白名单，SQL注入防护。
    link: /zh/modules/crud
  - icon: 🛡️
    title: Middleware 中间件
    details: Recovery、CORS、日志、超时、请求ID、请求体大小限制。
    link: /zh/modules/middleware
  - icon: 👥
    title: RBAC 权限控制
    details: 基于角色的访问控制，通配符权限，层级继承。
    link: /zh/modules/rbac
  - icon: ⏱️
    title: Rate Limit 限流
    details: 固定窗口、滑动窗口、令牌桶算法，LRU 淘汰。
    link: /zh/modules/ratelimit
  - icon: ✅
    title: Validation 验证
    details: 链式请求验证，正则DoS防护，panic恢复。
    link: /zh/modules/validation
  - icon: 📁
    title: File Upload 文件上传
    details: 安全文件上传，魔数校验，路径遍历防护。
    link: /zh/modules/file-upload
  - icon: 💚
    title: Health 健康检查
    details: 存活/就绪探针，SSRF防护。
    link: /zh/modules/health
  - icon: 💾
    title: Cache 缓存
    details: 通用 TTL + LRU 内存缓存，后台清理。
    link: /zh/modules/cache
  - icon: 📄
    title: Pagination 分页
    details: 偏移/游标分页，Base64校验。
    link: /zh/modules/pagination
  - icon: 📧
    title: Mail 邮件
    details: SMTP 邮件 + 模板，头部注入防护，异步队列。
    link: /zh/modules/mail
---

## 快速开始

```bash
# 复制模块到你的项目
cp -r registry/auth/src/go/* yourproject/internal/auth/

# 修改配置
# 编辑 config.go: 设置 JWT secret、数据库URL等

# 实现 store 接口
# type UserStore interface { ... }

# 注册路由
# 参考 registry/auth/examples/gin/main.go
```

## 为什么复制粘贴？

后端模块在不同项目间共享 80% 的骨架代码。与其安装框架，不如复制生产就绪的模块，拥有每一行代码。

- **代码所有权** — 复制后每一行都是你的
- **显式依赖** — 默认仅使用标准库；安全例外会声明
- **安全优先** — 内置输入验证、限流、注入防护
- **AI友好** — `__llms__.md` 文件让 AI 快速理解模块
