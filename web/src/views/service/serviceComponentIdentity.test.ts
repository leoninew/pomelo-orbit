import { describe, expect, it } from 'vitest';
import { serviceIdentityDraft, serviceIdentityPayload } from './serviceComponentIdentity';

describe('service component identity', () => {
  const declaration = { user: '1000:1000', group_add: ['988'] };

  it('keeps inheritance separate from image defaults', () => {
    const inherited = serviceIdentityDraft(declaration, { user: undefined, group_add: undefined });
    expect(inherited.user.base).toBe('1000:1000');
    expect(serviceIdentityPayload(inherited.user, inherited.group_add)).toEqual({
      user: undefined,
      group_add: undefined,
    });
    const cleared = serviceIdentityDraft(declaration, { user: '', group_add: { values: [] } });
    expect(serviceIdentityPayload(cleared.user, cleared.group_add)).toEqual({
      user: '',
      group_add: { values: [] },
    });
    cleared.user.overridden = false;
    cleared.group_add.overridden = false;
    expect(serviceIdentityPayload(cleared.user, cleared.group_add)).toEqual({
      user: undefined,
      group_add: undefined,
    });
  });

  it('replaces the complete supplementary group list', () => {
    const draft = serviceIdentityDraft(declaration, {
      user: '2000:2000',
      group_add: { values: ['999'] },
    });
    draft.group_add.value = '999, docker';
    expect(serviceIdentityPayload(draft.user, draft.group_add)).toEqual({
      user: '2000:2000',
      group_add: { values: ['999', 'docker'] },
    });
    draft.group_add.value = '999,,docker';
    expect(() => serviceIdentityPayload(draft.user, draft.group_add)).toThrow();
  });
});
