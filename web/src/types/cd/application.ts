// 应用相关
export interface Application {
  id: string;
  name: string;
  code: string;
  image_pull_policy: string;
  route_managed: boolean;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ApplicationCreateReq {
  name: string;
  code: string;
  image_pull_policy?: string;
  route_managed: boolean;
}

export interface ApplicationUpdateReq {
  name?: string;
  image_pull_policy?: string;
  route_managed?: boolean;
}

export interface ApplicationConfigFilePayload {
  path: string;
  content?: string;
}

export interface ApplicationServiceConfigPayload {
  service_name: string;
  image?: string | null;
  environment?: string | null;
  volumes?: string | null;
}

export interface ApplicationRoutePayload {
  service_name: string;
  domain: string;
  port: number;
}

export interface ApplicationExportConfigFileResp {
  path: string;
  content: string;
}

export interface ApplicationServiceConfigExportResp {
  service_name: string;
  image: string | null;
  environment: string | null;
  volumes: string | null;
}

export interface ApplicationExportRouteResp {
  service_name: string;
  domain: string;
  port: number;
}

export interface ApplicationExportResp {
  version: string;
  name: string;
  code: string;
  image_pull_policy: string;
  route_managed: boolean;
  config_files: ApplicationExportConfigFileResp[];
  service_configs: ApplicationServiceConfigExportResp[];
  routes: ApplicationExportRouteResp[];
}

export interface ApplicationImportReq {
  version?: string;
  name: string;
  code: string;
  image_pull_policy?: string;
  route_managed?: boolean;
  config_files?: ApplicationConfigFilePayload[];
  service_configs?: ApplicationServiceConfigPayload[];
  routes?: ApplicationRoutePayload[];
}

export interface ConfigFile {
  id: string;
  path: string;
  created_at: string;
  updated_at: string;
}

export interface ApplicationRoute {
  id: string;
  service_name: string;
  domain: string;
  port: number;
  created_at: string;
  updated_at: string;
}

export interface ApplicationServiceConfig {
  service_name: string;
  default_domain: string;
  default_port: number;
  base_image: string | null;
  image: string | null;
  config_id: string | null;
  created_at: string | null;
  updated_at: string | null;
}

export interface ComposeServiceResp {
  service_name: string;
  default_domain: string;
  default_port: number;
}

export interface ApplicationFormState {
  name: string;
  code: string;
  image_pull_policy: string;
  route_managed: boolean;
}

export interface ApplicationImportState extends ApplicationFormState {
  version: string | undefined;
  config_files: NonNullable<ApplicationImportReq['config_files']>;
  service_configs: NonNullable<ApplicationImportReq['service_configs']>;
  routes: NonNullable<ApplicationImportReq['routes']>;
}
