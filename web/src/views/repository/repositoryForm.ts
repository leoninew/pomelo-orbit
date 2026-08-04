export type RepositoryFormFields = 'name' | 'code' | 'repository_url';

export type RepositoryFormErrors = Partial<Record<RepositoryFormFields, string>>;

export type RepositoryFormInput = {
  name: string;
  code?: string;
  repository_type: string;
  repository_url: string;
};

export type RepositoryFormFeedback =
  { kind: 'field'; errors: RepositoryFormErrors } | { kind: 'form'; message: string };

export function validateRepositoryForm(
  form: RepositoryFormInput,
  options: { requireCode: boolean }
): RepositoryFormErrors {
  const errors: RepositoryFormErrors = {};

  if (!form.name.trim()) {
    errors.name = '请输入名称';
  }
  if (options.requireCode && (!form.code?.trim() || !/^[a-z0-9_-]+$/.test(form.code.trim()))) {
    errors.code = '编码只能包含小写字母、数字、连字符或下划线';
  }
  if (!form.repository_url.trim()) {
    errors.repository_url =
      form.repository_type === 'local_directory' ? '请输入本地目录' : '请输入仓库地址';
  }

  return errors;
}

export function repositoryFormFeedback(error: unknown): RepositoryFormFeedback | undefined {
  const apiError = error as { code?: unknown; kind?: unknown; message?: unknown } | undefined;
  if (
    !apiError ||
    apiError.kind !== 'api' ||
    (apiError.code !== 'validation_failed' && apiError.code !== 'conflict') ||
    typeof apiError.message !== 'string'
  ) {
    return undefined;
  }

  const message = apiError.message || '提交内容无效';
  if (
    /^Invalid local source path:|^local directory repositories?|^repository_url\b/i.test(message)
  ) {
    return { kind: 'field', errors: { repository_url: message } };
  }
  if (/^Repository code '\S+' already exists$/i.test(message)) {
    return { kind: 'field', errors: { code: message } };
  }
  if (/\brepository_url\b/i.test(message)) {
    return { kind: 'field', errors: { repository_url: message } };
  }

  return { kind: 'form', message };
}
