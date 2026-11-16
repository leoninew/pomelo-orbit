// @vitest-environment happy-dom
import { createApp, h } from 'vue';
import { describe, expect, it } from 'vitest';
import MarkdownContent from '@/components/ui/MarkdownContent.vue';

describe('MarkdownContent', () => {
  it('renders Markdown while removing unsafe HTML', () => {
    const target = document.createElement('div');
    const app = createApp({
      render: () =>
        h(MarkdownContent, {
          content: '**Hello** <img src="x" onerror="alert(1)"><script>alert(1)</script>',
        }),
    });
    document.body.append(target);
    app.mount(target);

    try {
      expect(target.querySelector('strong')?.textContent).toBe('Hello');
      expect(target.querySelector('img')?.hasAttribute('onerror')).toBe(false);
      expect(target.querySelector('script')).toBeNull();
    } finally {
      app.unmount();
      target.remove();
    }
  });
});
