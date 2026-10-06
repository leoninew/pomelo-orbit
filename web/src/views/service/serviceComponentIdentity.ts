import type {
  ServiceComponentDefinitionResp,
  ServiceComponentResp,
  ServiceComponentOverlayUpdateReq,
} from '@/gen/proto/orbit/v1/service/service';
import { parseGroupAdd } from '@/utils/componentIdentity';

export type IdentityFieldState = { base: string; value: string; overridden: boolean };

export function serviceIdentityDraft(
  declaration: Pick<ServiceComponentDefinitionResp, 'user' | 'group_add'>,
  component: Pick<ServiceComponentResp, 'user' | 'group_add'>
) {
  return {
    user: {
      base: declaration.user ?? '',
      value: component.user ?? '',
      overridden: component.user !== undefined,
    },
    group_add: {
      base: declaration.group_add.join('\n'),
      value: component.group_add?.values.join('\n') ?? '',
      overridden: component.group_add !== undefined,
    },
  };
}

export function serviceIdentityPayload(
  user: IdentityFieldState,
  groups: IdentityFieldState
): Pick<ServiceComponentOverlayUpdateReq, 'user' | 'group_add'> {
  const values = groups.overridden ? parseGroupAdd(groups.value) : [];
  if (values === null) {
    throw new Error('Invalid group_add');
  }
  return {
    user: user.overridden ? user.value : undefined,
    group_add: groups.overridden ? { values } : undefined,
  };
}
