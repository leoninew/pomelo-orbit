/** Per-project last selected application on the Versions page. */
export function applicationVersionsApplicationIdKey(projectId: string): string {
  return `pomelo_orbit_application_versions_application_id:${projectId}`;
}
