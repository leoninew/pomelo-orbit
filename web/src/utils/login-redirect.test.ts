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
    { path: '/gateway', name: 'Gateway', component: {} },
  ],
});

describe('resolveLoginRedirect', () => {
  it('preserves a registered protected route and its query and hash', () => {
    expect(resolveLoginRedirect(router, '/gateway?status=healthy#summary')).toBe(
      '/gateway?status=healthy#summary'
    );
  });

  it('unwraps a nested login redirect to the original protected route', () => {
    expect(
      resolveLoginRedirect(router, '/login?redirect=/login?redirect=/login?redirect=/gateway')
    ).toBe('/gateway');
  });

  it.each([
    ['/login'],
    ['/login?redirect=/login'],
    ['/google/callback?code=code-1'],
    ['/missing'],
    ['https://example.com/gateway'],
    ['//example.com/gateway'],
    ['/\\example.com/gateway'],
  ])('falls back to the home route for %s', (redirect) => {
    expect(resolveLoginRedirect(router, redirect)).toBe('/');
  });
});
