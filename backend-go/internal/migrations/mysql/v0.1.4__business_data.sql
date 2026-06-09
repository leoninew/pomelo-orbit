-- v0.1.4: Initial business data aligned with backend v0.7.2

INSERT IGNORE INTO application (
    id, name, code, image_pull_policy, status, route_managed, project_id, created_at, updated_at
) VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KVM',
    'Traefik',
    'traefik',
    'missing',
    'stopped',
    0,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO application (
    id, name, code, image_pull_policy, status, route_managed, project_id, created_at, updated_at
) VALUES (
    '01KN8CG4A5S4VVH6NKNJF4F9NJ',
    'FileBrowser',
    'filebrowser',
    'missing',
    'stopped',
    1,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO application_config_file (
    id, application_id, path, content, created_at, updated_at
) VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KVR',
    '01KKX2YNPF6VJ9N7QYCWG61KVM',
    '.env',
    'TRAEFIK_LOG_LEVEL=INFO
',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO application_config_file (
    id, application_id, path, content, created_at, updated_at
) VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KVQ',
    '01KKX2YNPF6VJ9N7QYCWG61KVM',
    'data/traefik.yml.jinja',
    'api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

{% if cert.letsencrypt.enabled %}
certificatesResolvers:
  letsencrypt:
    acme:
      email: {{ cert.letsencrypt.email }}
      storage: /etc/traefik/acme.json
      {% if cert.letsencrypt.challenge == "http" %}
      httpChallenge:
        entryPoint: web
      {% elif cert.letsencrypt.challenge == "dns" %}
      dnsChallenge:
        provider: {{ cert.letsencrypt.dns_provider }}
      {% endif %}
{% endif %}

providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: traefik

log:
  level: INFO
',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO application_config_file (
    id, application_id, path, content, created_at, updated_at
) VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KVN',
    '01KKX2YNPF6VJ9N7QYCWG61KVM',
    'docker-compose.yml.jinja',
    'services:
  traefik:
    image: traefik:3
    container_name: traefik
    restart: unless-stopped
    entrypoint: ["sh", "-c", "chmod 600 /etc/traefik/acme.json && exec traefik"]
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"
    env_file: .env
    networks:
      - traefik
    volumes:
      - {{ app.physical_app_dir }}/data/traefik.yml:/etc/traefik/traefik.yml:ro
      - {{ app.physical_app_dir }}/data/dynamic:/etc/traefik/dynamic:ro
      - {{ app.physical_app_dir }}/data/certs:/etc/traefik/certs:ro
      - {{ app.physical_app_dir }}/data/acme.json:/etc/traefik/acme.json
      - /var/run/docker.sock:/var/run/docker.sock:ro
    labels:
      {% if cert.letsencrypt.enabled %}
      # 1. HTTPS 主路由（Traefik 面板）
      - traefik.enable=true
      - traefik.http.routers.traefik-https.rule=Host(`{{app.code}}.{{ config.domain_suffix }}`)
      - traefik.http.routers.traefik-https.entrypoints=websecure
      - traefik.http.routers.traefik-https.tls=true
      - traefik.http.routers.traefik-https.tls.certresolver=letsencrypt
      - traefik.http.routers.traefik-https.service=api@internal

      {% else %}
      # 2. HTTP 路由
      - traefik.enable=true
      - traefik.http.routers.traefik-http.rule=Host(`{{app.code}}.{{ config.domain_suffix }}`)
      - traefik.http.routers.traefik-http.entrypoints=web
      - traefik.http.routers.traefik-http.service=api@internal
      {% endif %}

networks:
  traefik:
    name: traefik
    driver: bridge
',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO application_config_file (
    id, application_id, path, content, created_at, updated_at
) VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KVP',
    '01KKX2YNPF6VJ9N7QYCWG61KVM',
    'init.sh',
    '#!/bin/bash
set -e

echo "Initializing Traefik..."

# 创建应用数据目录
mkdir -p data/dynamic
mkdir -p data/certs

# 仅在不存在时创建 acme.json（避免覆盖已有证书）
if [ ! -f data/acme.json ]; then
  touch data/acme.json
fi

echo "Traefik initialized"
',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO application_config_file (
    id, application_id, path, content, created_at, updated_at
) VALUES (
    '01KN8CG4A5PBH7TDXAQ0MD5Z34',
    '01KN8CG4A5S4VVH6NKNJF4F9NJ',
    'docker-compose.yml.jinja',
    'services:
  filebrowser:
    image: filebrowser/filebrowser
    container_name: filebrowser
    restart: unless-stopped
    environment:
      - PUID=1000
      - PGID=1000
    networks:
      - traefik
    volumes:
      - {{ app.physical_dir }}:/srv
      - {{ app.physical_app_dir }}/data:/database

networks:
  traefik:
    external: true
',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO application_config_file (
    id, application_id, path, content, created_at, updated_at
) VALUES (
    '01KN8CG4A6PBH7TDXAQ0MD5Z35',
    '01KN8CG4A5S4VVH6NKNJF4F9NJ',
    'init.sh',
    '#!/bin/bash
set -e

echo "Initializing FileBrowser..."

mkdir -p data
chown -R 1000:100 data
chmod 755 data

echo "FileBrowser initialized"
',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO route (
    id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, project_id, created_at, updated_at
) VALUES (
    '01KRRKJAHSNF4FQKSM9D30K6S2',
    'example-route',
    'example.com',
    '/',
    'http://127.0.0.1:8081',
    0,
    0,
    NULL,
    NULL,
    'manual',
    NULL,
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO repository (
    id, name, code, repository_url, git_credential_id, variable_overrides, default_branch, project_id, created_at, updated_at
) VALUES (
    '01KNNRBH52BQJYT9487B2H8N62',
    'golang/example',
    'golang-example',
    'https://github.com/golang/example',
    NULL,
    '[]',
    'master',
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO repository (
    id, name, code, repository_url, git_credential_id, variable_overrides, default_branch, project_id, created_at, updated_at
) VALUES (
    '01KP0JZFQQA2Z77FRRVH35BYF6',
    'docker/awesome-compose',
    'awesome-compose',
    'https://github.com/docker/awesome-compose',
    NULL,
    '[]',
    'master',
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO build_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01STAGE000000000000000CLONE',
    'git clone',
    'alpine/git',
    'set -e
# 强制全局关闭 SSL 验证
git config --global http.sslVerify "false"
git config --system http.sslVerify "false"
# 信任工作区目录
git config --global --add safe.directory /workspace
# 初始化仓库
git init
git remote remove origin 2>/dev/null || true
git remote add origin {{ repository_url }}
# 拉取代码
git fetch --depth=1 origin {{ repository_ref }}
git checkout -B {{ repository_ref }} FETCH_HEAD',
    NULL,
    '克隆代码仓库',
    8,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO build_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01KNRANZDR4PASATAXKTBBTRX9',
    'golang:1.23 test',
    'golang:1.23-alpine',
    'set -e
cd {{ working_dir | default(''.'') }}
go env -w GOPROXY=https://goproxy.cn,direct
go test ./...',
    NULL,
    '运行 Go 单元测试',
    3,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO build_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01KNRDSSJ7RNND7110175N4NR2',
    'golang:1.23 build',
    'golang:1.23-alpine',
    'set -e
cd {{ working_dir | default(''.'') }}
mkdir -p dist
go env -w GOPROXY=https://goproxy.cn,direct
go build -o dist/',
    NULL,
    '运行 Go 构建',
    5,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO build_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01KNRKNAHG3EBS07VBK2YY5ZQN',
    'golang:1.23 lint',
    'golang:1.23-alpine',
    'set -e
cd {{ working_dir | default(''.'') }}
go env -w GOPROXY=https://goproxy.cn,direct
go install golang.org/x/lint/golint@latest
golint ./...',
    NULL,
    '运行 Go 代码质量',
    3,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO build_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01KP0K4ZTV60PM2XDT11MTX2MY',
    'docker build',
    'docker:29.4',
    'set -e
cd {{ working_dir | default(''.'') }}
docker build -t {{ repository_code }}:{{ runtime_datetime }} -f {{ repository_dockerfile | default(''Dockerfile'') }} .
# docker push {{ repository_code }}:{{ runtime_datetime }}',
    '[{"type": "docker_image", "path": "{{ repository_code }}:{{ runtime_datetime }}", "name": "{{ repository_code }}"}]',
    '通用镜像构建+推送',
    16,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO pipeline_template (
    id, name, description, variable_declarations, version, project_id, created_at, updated_at
) VALUES (
    '01KNVEJPWVK757139NMNNNCEFE',
    'Go 构建流水线',
    '- test & lint
- build',
    '[{"name":"working_dir","description":"","default":".","value":"hello","secret":false,"source":"template_custom","editable":true}]',
    7,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO pipeline_template (
    id, name, description, variable_declarations, version, project_id, created_at, updated_at
) VALUES (
    '01KP0K6W1YW73REFQE8YAVTPMN',
    '通用容器镜像流水线',
    '- docker build',
    '[]',
    24,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCWNJVA1DM02TJXZ4STJCCK',
    '01KNVEJPWVK757139NMNNNCEFE',
    '01STAGE000000000000000CLONE',
    'git clone',
    8,
    '[]',
    0
);

INSERT IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCWNJVA1DM02TJXZ4STJCCM',
    '01KNVEJPWVK757139NMNNNCEFE',
    '01KNRANZDR4PASATAXKTBBTRX9',
    'golang:1.23 test',
    3,
    '["01STAGE000000000000000CLONE"]',
    1
);

INSERT IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCWNJVA1DM02TJXZ4STJCCN',
    '01KNVEJPWVK757139NMNNNCEFE',
    '01KNRKNAHG3EBS07VBK2YY5ZQN',
    'golang:1.23 lint',
    3,
    '["01STAGE000000000000000CLONE"]',
    2
);

INSERT IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCWNJVA1DM02TJXZ4STJCCP',
    '01KNVEJPWVK757139NMNNNCEFE',
    '01KNRDSSJ7RNND7110175N4NR2',
    'golang:1.23 build',
    5,
    '["01KNRANZDR4PASATAXKTBBTRX9"]',
    3
);

INSERT IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCYJJF8GNWTA76CE2M4QXAZ',
    '01KP0K6W1YW73REFQE8YAVTPMN',
    '01STAGE000000000000000CLONE',
    'git clone',
    8,
    '[]',
    0
);

INSERT IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCYJJF8GNWTA76CE2M4QXB0',
    '01KP0K6W1YW73REFQE8YAVTPMN',
    '01KP0K4ZTV60PM2XDT11MTX2MY',
    'docker build',
    16,
    '["01STAGE000000000000000CLONE"]',
    1
);
