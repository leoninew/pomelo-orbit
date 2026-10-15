type SshKeyCommentInput = {
  keyCreatedAt: string | undefined;
  site: string;
  sourceHost: string | undefined;
  projectName: string | null;
  projectId: string;
  environmentId: string;
};

export function buildSshKeyComment(input: SshKeyCommentInput): string | null {
  if (!input.keyCreatedAt?.trim() || !input.sourceHost?.trim()) {
    return null;
  }
  const fields = {
    key_created_at: input.keyCreatedAt,
    site: input.site,
    source_host: input.sourceHost,
    project: input.projectName,
    project_id: input.projectId,
    environment_id: input.environmentId,
  };
  return `# Pomelo Orbit | ${Object.entries(fields)
    .map(([name, value]) => `${name}=${JSON.stringify(value)}`)
    .join(' | ')}`;
}
