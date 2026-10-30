import { defineConfig } from 'vitepress';

export default defineConfig({
  srcDir: './handbook',
  base: '/pomelo-orbit/',
  lang: 'zh-CN',
  title: 'Pomelo Orbit',
  description: 'Pomelo Orbit 开发文档',
  themeConfig: {
    nav: [
      { text: '首页', link: '/' },
      { text: '开发入门', link: '/getting-started/development' },
    ],
    sidebar: [
      {
        text: '开始使用',
        items: [{ text: '开发入门', link: '/getting-started/development' }],
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
