// @vitest-environment happy-dom
import { createMemoryHistory, createRouter } from 'vue-router';
import { describe, expect, it } from 'vitest';
import { resolveLoginRedirect } from './login-redirect';

const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/', name: 'Home', component: {} },
    { path: '/login', name: 'Login', component: {}, meta: { public: true } },
    { path: '/google/callback', name: 'GoogleCallback', component: {}, meta: { public: true } },
    { path: '/gateways', name: 'Gateways', component: {} },
    { path: '/mcp-authorize', name: 'MCPAuthorize', component: {} },
  ],
});

describe('resolveLoginRedirect', () => {
  it('preserves a registered protected route and its query and hash', () => {
    expect(resolveLoginRedirect(router, '/gateways?status=healthy#summary')).toBe(
      '/gateways?status=healthy#summary'
    );
  });

  it('unwraps a nested login redirect to the original protected route', () => {
    expect(
      resolveLoginRedirect(router, '/login?redirect=/login?redirect=/login?redirect=/gateways')
    ).toBe('/gateways');
  });

  it('preserves the protected MCP authorization route and its callback query', () => {
    expect(
      resolveLoginRedirect(
        router,
        '/mcp-authorize?callback=http%3A%2F%2F127.0.0.1%3A48123%2Fmcp%2Fcallback&state=state-1'
      )
    ).toBe(
      '/mcp-authorize?callback=http%3A%2F%2F127.0.0.1%3A48123%2Fmcp%2Fcallback&state=state-1'
    );
  });

  it.each([
    ['/login'],
    ['/login?redirect=/login'],
    ['/google/callback?code=code-1'],
    ['/missing'],
    ['https://example.com/gateways'],
    ['//example.com/gateways'],
    ['/\\example.com/gateways'],
  ])('falls back to the home route for %s', (redirect) => {
    expect(resolveLoginRedirect(router, redirect)).toBe('/');
  });
});
