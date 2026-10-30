import { defineConfig } from 'vitepress';

export default defineConfig({
  srcDir: './handbook',
  base: '/pomelo-orbit/',
  lang: 'zh-CN',
  title: 'Pomelo Orbit',
  description: 'Pomelo Orbit 使用手册：项目配置、构建部署、网络入口与实例维护',
  themeConfig: {
    nav: [
      { text: '首页', link: '/' },
      { text: '项目概览', link: '/getting-started/overview' },
      { text: '核心概念', link: '/concepts/core-model' },
      { text: '使用指南', link: '/guides/projects-environments' },
      { text: '运维与排障', link: '/operations/configuration' },
      { text: '进阶使用', link: '/reference/mcp-cli' },
    ],
    sidebar: [
      {
        text: '开始使用',
        items: [
          { text: '项目概览', link: '/getting-started/overview' },
          { text: '自托管安装', link: '/getting-started/installation' },
          { text: '首个应用部署', link: '/getting-started/first-deployment' },
        ],
      },
      {
        text: '核心概念',
        items: [{ text: '核心概念', link: '/concepts/core-model' }],
      },
      {
        text: '使用指南',
        items: [
          { text: '项目与执行环境', link: '/guides/projects-environments' },
          { text: '仓库与仓库凭据', link: '/guides/repositories-credentials' },
          { text: '流水线与制品', link: '/guides/pipelines-artifacts' },
          { text: '应用、版本与服务', link: '/guides/applications-services' },
          { text: '网关、DNS 与路由', link: '/guides/gateway-dns-routes' },
          { text: '用户、权限与访问令牌', link: '/guides/users-access' },
          { text: '部署对话', link: '/guides/deployment-dialogue' },
        ],
      },
      {
        text: '运维与排障',
        items: [
          { text: '配置与密钥', link: '/operations/configuration' },
          { text: '数据保留与升级', link: '/operations/data-upgrades' },
          { text: '故障定位', link: '/operations/troubleshooting' },
        ],
      },
      {
        text: '进阶使用',
        items: [
          { text: 'MCP 与 CLI', link: '/reference/mcp-cli' },
          { text: '变量与 Compose 约束', link: '/reference/variables-compose' },
        ],
      },
    ],
    socialLinks: [{ icon: 'github', link: 'https://github.com/leoninew/pomelo-orbit' }],
    outline: { label: '本页目录' },
    docFooter: { prev: '上一篇', next: '下一篇' },
    returnToTopLabel: '返回顶部',
    sidebarMenuLabel: '目录',
    darkModeSwitchLabel: '主题',
    skipToContentLabel: '跳转至正文',
  },
});
