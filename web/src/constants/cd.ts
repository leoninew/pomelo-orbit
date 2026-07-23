/** Per-project last selected application on the Versions page. */
export function cdVersionsApplicationIdKey(projectId: string): string {
  return `pomelo_orbit_cd_versions_application_id:${projectId}`;
}
