import type {
  VariableDeclarationReq,
  VariableDeclarationResp,
} from '@/gen/proto/orbit/v1/common/common';

type VariableDeclarationRequestSource = Pick<
  VariableDeclarationResp,
  'name' | 'description' | 'default' | 'value' | 'secret' | 'source' | 'editable' | 'stage_id'
>;

type VariableValueSource = Pick<VariableDeclarationResp, 'default' | 'value'> & {
  stage_defaults?: VariableDeclarationResp['stage_defaults'];
};

export function effectiveVariableValue(variable: VariableValueSource) {
  if (hasVariableValue(variable.value)) {
    return variable.value;
  }
  if (hasVariableValue(variable.default)) {
    return variable.default;
  }
  return variable.stage_defaults?.find((item) => hasVariableValue(item.default))?.default;
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
