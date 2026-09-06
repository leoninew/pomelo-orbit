import { describe, expect, it } from 'vitest';
import type { VariableDeclarationResp } from '@/gen/proto/orbit/v1/common/common';
import { toVariableDeclarationRequest } from './variableDeclaration';

describe('toVariableDeclarationRequest', () => {
  it('keeps only fields accepted by the variable declaration request contract', () => {
    const response: VariableDeclarationResp = {
      name: 'working_dir',
      description: 'build directory',
      default: null,
      value: 'web',
      secret: false,
      source: 'pipeline_custom',
      editable: true,
      stage_defaults: [{ stage_id: 'frontend', stage_name: 'Frontend', default: 'web' }],
      stage_id: 'frontend',
      stage_name: 'Frontend',
    };

    expect(toVariableDeclarationRequest(response)).toEqual({
      name: 'working_dir',
      description: 'build directory',
      default: null,
      value: 'web',
      secret: false,
      source: 'pipeline_custom',
      editable: true,
      stage_id: 'frontend',
    });
  });
});
