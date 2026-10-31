import { defineConfig } from 'vitepress';

export default defineConfig({
  srcDir: './handbook',
  base: '/pomelo-orbit/',
  lang: 'zh-CN',
  title: 'Pomelo Orbit',
  description: 'Pomelo Orbit 使用手册：项目配置、构建部署、网络入口与实例维护',
  markdown: {
    config(md) {
      const fence = md.renderer.rules.fence!;
      md.renderer.rules.fence = (tokens, index, options, env, self) => {
        if (tokens[index].info.trim() === 'mermaid') {
          return `<MermaidDiagram source="${encodeURIComponent(tokens[index].content)}" />`;
        }
        return fence(tokens, index, options, env, self);
      };
    },
  },
  themeConfig: {
    nav: [
      { text: '手册', link: '/' },
      { text: '快速开始', link: '/getting-started/first-deployment' },
      { text: '部署应用', link: '/guides/applications' },
      { text: '实例维护', link: '/operations/configuration' },
    ],
    sidebar: [
      {
        text: '开始使用',
        items: [
          { text: '项目概览', link: '/getting-started/overview' },
          { text: '本地开发', link: '/getting-started/development' },
          { text: '自托管运行', link: '/getting-started/installation' },
          { text: '部署首个应用', link: '/getting-started/first-deployment' },
          { text: '资源关系', link: '/concepts/core-model' },
        ],
      },
      {
        text: '项目与访问',
        items: [
          { text: '项目', link: '/guides/projects' },
          { text: '执行环境', link: '/guides/environments' },
          { text: '用户', link: '/guides/users' },
          { text: '角色与权限', link: '/guides/roles' },
          { text: '访问令牌', link: '/guides/access-tokens' },
        ],
      },
      {
        text: '持续集成',
        items: [
          { text: '代码仓库', link: '/guides/repositories' },
          { text: '仓库凭据', link: '/guides/repository-credentials' },
          { text: '阶段', link: '/guides/stages' },
          { text: '流水线', link: '/guides/pipelines' },
          { text: '运行记录', link: '/guides/runs' },
          { text: '制品', link: '/guides/artifacts' },
          { text: '构建变量', link: '/reference/ci-variables' },
        ],
      },
      {
        text: '持续部署',
        items: [
          { text: '应用', link: '/guides/applications' },
          { text: '版本', link: '/guides/versions' },
          { text: '组件', link: '/guides/components' },
          { text: '服务', link: '/guides/services' },
          { text: '部署记录', link: '/guides/deployments' },
          { text: '服务变量', link: '/reference/service-variables' },
          { text: 'Compose 生成规则', link: '/reference/compose' },
        ],
      },
      {
        text: '网络接入',
        items: [
          { text: '网关', link: '/guides/gateway' },
          { text: 'DNS 解析', link: '/guides/dns' },
          { text: '自定义路由', link: '/guides/routes' },
          { text: 'HTTPS 证书', link: '/guides/certificates' },
        ],
      },
      {
        text: '自动化',
        items: [
          { text: '部署对话', link: '/guides/deployment-dialogue' },
          { text: 'MCP', link: '/reference/mcp' },
          { text: '环境 CLI', link: '/reference/cli' },
        ],
      },
      {
        text: '实例维护',
        items: [
          { text: '系统配置', link: '/operations/configuration' },
          { text: '密钥管理', link: '/operations/secrets' },
          { text: '备份与恢复', link: '/operations/backup' },
          { text: '升级', link: '/operations/upgrade' },
          { text: '故障定位', link: '/operations/troubleshooting' },
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
