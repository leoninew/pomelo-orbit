import type {
  VariableDeclarationReq,
  VariableConfigurationResp,
} from '@/gen/proto/orbit/v1/common/common';

type VariableDeclarationRequestSource = Pick<
  VariableDeclarationReq,
  'name' | 'description' | 'default' | 'value' | 'secret' | 'source' | 'editable' | 'stage_id'
>;

export function effectiveVariableValue(
  configuration?: Pick<VariableConfigurationResp, 'default' | 'value'>
) {
  const value = configuration?.value;
  if (hasVariableValue(value)) {
    return value;
  }
  const defaultValue = configuration?.default;
  if (hasVariableValue(defaultValue)) {
    return defaultValue;
  }
}

function hasVariableValue(value: unknown) {
  if (value === null || value === undefined) {
    return false;
  }
  return typeof value !== 'string' || value.trim().length > 0;
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
