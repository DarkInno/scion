import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/scion/',
  title: 'Scion',
  description: 'Copy-paste Go backend source templates — security-first, AI-friendly',
  
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/scion/logo.svg' }]
  ],

  locales: {
    root: {
      label: 'English',
      lang: 'en',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/guide/getting-started' },
          { text: 'Modules', link: '/modules/' },
          { text: 'GitHub', link: 'https://github.com/DarkInno/scion' }
        ],
        sidebar: {
          '/guide/': [
            {
              text: 'Introduction',
              items: [
                { text: 'Getting Started', link: '/guide/getting-started' },
                { text: 'Why Copy-Paste?', link: '/guide/why-copy-paste' },
                { text: 'Security Design', link: '/guide/security' }
              ]
            }
          ],
          '/modules/': [
            {
              text: 'Modules',
              items: [
                { text: 'Overview', link: '/modules/' },
                { text: 'Auth', link: '/modules/auth' },
                { text: 'CRUD', link: '/modules/crud' },
                { text: 'Database', link: '/modules/database' },
                { text: 'Middleware', link: '/modules/middleware' },
                { text: 'RBAC', link: '/modules/rbac' },
                { text: 'Rate Limit', link: '/modules/ratelimit' },
                { text: 'Validation', link: '/modules/validation' },
                { text: 'File Upload', link: '/modules/file-upload' },
                { text: 'Health', link: '/modules/health' },
                { text: 'Cache', link: '/modules/cache' },
                { text: 'Pagination', link: '/modules/pagination' },
                { text: 'Mail', link: '/modules/mail' }
              ]
            }
          ]
        },
        outline: {
          level: [2, 3],
          label: 'On this page'
        }
      }
    },
    zh: {
      label: '中文',
      lang: 'zh-CN',
      title: 'Scion',
      description: '复制粘贴 Go 后端源码模板 — 安全优先、AI友好',
      themeConfig: {
        nav: [
          { text: '指南', link: '/zh/guide/getting-started' },
          { text: '模块', link: '/zh/modules/' },
          { text: 'GitHub', link: 'https://github.com/DarkInno/scion' }
        ],
        sidebar: {
          '/zh/guide/': [
            {
              text: '介绍',
              items: [
                { text: '快速开始', link: '/zh/guide/getting-started' },
                { text: '为什么复制粘贴？', link: '/zh/guide/why-copy-paste' },
                { text: '安全设计', link: '/zh/guide/security' }
              ]
            }
          ],
          '/zh/modules/': [
            {
              text: '模块',
              items: [
                { text: '概览', link: '/zh/modules/' },
                { text: 'Auth 认证', link: '/zh/modules/auth' },
                { text: 'CRUD 增删改查', link: '/zh/modules/crud' },
                { text: 'Database 数据库', link: '/zh/modules/database' },
                { text: 'Middleware 中间件', link: '/zh/modules/middleware' },
                { text: 'RBAC 权限控制', link: '/zh/modules/rbac' },
                { text: 'Rate Limit 限流', link: '/zh/modules/ratelimit' },
                { text: 'Validation 验证', link: '/zh/modules/validation' },
                { text: 'File Upload 文件上传', link: '/zh/modules/file-upload' },
                { text: 'Health 健康检查', link: '/zh/modules/health' },
                { text: 'Cache 缓存', link: '/zh/modules/cache' },
                { text: 'Pagination 分页', link: '/zh/modules/pagination' },
                { text: 'Mail 邮件', link: '/zh/modules/mail' }
              ]
            }
          ]
        },
        outline: {
          level: [2, 3],
          label: '页面导航'
        }
      }
    }
  },

  themeConfig: {
    logo: '/logo.svg',
    
    socialLinks: [
      { icon: 'github', link: 'https://github.com/DarkInno/scion' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: '© 2026 DarkInno'
    },

    search: {
      provider: 'local'
    }
  }
})
