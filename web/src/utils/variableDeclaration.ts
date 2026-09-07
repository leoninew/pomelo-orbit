import type {
  VariableDeclarationReq,
  VariableDeclarationResp,
} from '@/gen/proto/orbit/v1/common/common';

type VariableDeclarationRequestSource = Pick<
  VariableDeclarationResp,
  'name' | 'description' | 'default' | 'value' | 'secret' | 'source' | 'editable' | 'stage_id'
>;

type VariableValueSource = Pick<VariableDeclarationResp, 'default' | 'value'>;

export function effectiveVariableValue(variable: VariableValueSource) {
  if (typeof variable.value === 'string' && variable.value.trim() === '') {
    return variable.default;
  }
  return variable.value ?? variable.default;
}

export function toVariableDeclarationRequest(
  variable: VariableDeclarationRequestSource
): VariableDeclarationReq {
  return {
    name: variable.name,
    description: variable.description,
    default: variable.default,
    value: variable.value,
    secret: variable.secret,
    source: variable.source,
    editable: variable.editable,
    stage_id: variable.stage_id,
  };
}
