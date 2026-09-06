import type {
  VariableDeclarationReq,
  VariableDeclarationResp,
} from '@/gen/proto/orbit/v1/common/common';

type VariableDeclarationRequestSource = Pick<
  VariableDeclarationResp,
  'name' | 'description' | 'default' | 'value' | 'secret' | 'source' | 'editable' | 'stage_id'
>;

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
