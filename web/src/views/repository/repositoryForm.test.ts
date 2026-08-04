import { describe, expect, it } from 'vitest';
import { repositoryFormFeedback, validateRepositoryForm } from './repositoryForm';

function apiError(message: string, code: 'validation_failed' | 'conflict' | 'network_error') {
  return { message, code, kind: code === 'network_error' ? 'network' : 'api' };
}

describe('validateRepositoryForm', () => {
  it('assigns a missing local directory to repository_url', () => {
    expect(
      validateRepositoryForm(
        {
          name: 'Repository',
          code: 'repository',
          repository_type: 'local_directory',
          repository_url: '  ',
        },
        { requireCode: true }
      )
    ).toEqual({ repository_url: '请输入本地目录' });
  });

  it('accepts underscores in repository codes', () => {
    expect(
      validateRepositoryForm(
        {
          name: 'Repository',
          code: 'repository_code',
          repository_type: 'remote_git',
          repository_url: 'https://example.test/repository.git',
        },
        { requireCode: true }
      )
    ).toEqual({});
  });
});

describe('repositoryFormFeedback', () => {
  it('assigns local source validation to repository_url', () => {
    expect(
      repositoryFormFeedback(
        apiError('Invalid local source path: directory does not exist', 'validation_failed')
      )
    ).toEqual({
      kind: 'field',
      errors: { repository_url: 'Invalid local source path: directory does not exist' },
    });
  });

  it('assigns duplicate repository codes to code', () => {
    expect(
      repositoryFormFeedback(apiError("Repository code 'repository' already exists", 'conflict'))
    ).toEqual({
      kind: 'field',
      errors: { code: "Repository code 'repository' already exists" },
    });
  });

  it('keeps other validation failures inside the dialog', () => {
    expect(
      repositoryFormFeedback(apiError('Unsupported repository_type', 'validation_failed'))
    ).toEqual({ kind: 'form', message: 'Unsupported repository_type' });
  });

  it('does not turn network failures into form errors', () => {
    expect(
      repositoryFormFeedback(apiError('网络连接失败，请检查网络设置', 'network_error'))
    ).toBeUndefined();
  });
});
