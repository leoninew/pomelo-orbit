-- v0.5.2: Traefik 应用（含 Let's Encrypt 支持）
-- 包含 v0.4.2 和 v0.4.5 的内容

-- Traefik 应用（bridge 网络模式，通用）
INSERT INTO application (id, name, code, image_pull_policy, status, created_at, updated_at)
VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KVM',
    'Traefik',
    'traefik',
    'missing',
    'stopped',
    datetime('now'),
    datetime('now')
);

-- Traefik docker-compose.yml.jinja（支持 Let's Encrypt）
INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
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
    datetime('now'),
    datetime('now')
);

-- Traefik init.sh
INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
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
    datetime('now'),
    datetime('now')
);

-- Traefik traefik.yml.jinja (Jinja 模板)
INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
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
    datetime('now'),
    datetime('now')
);

-- Traefik .env
INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KVR',
    '01KKX2YNPF6VJ9N7QYCWG61KVM',
    '.env',
    'TRAEFIK_LOG_LEVEL=INFO
',
    datetime('now'),
    datetime('now')
);
