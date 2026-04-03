-- v0.5.3: FileBrowser 应用

INSERT INTO application (id, name, code, image_pull_policy, status, created_at, updated_at)
VALUES (
    '01KN8CG4A5S4VVH6NKNJF4F9NJ',
    'FileBrowser',
    'filebrowser',
    'missing',
    'stopped',
    datetime('now'),
    datetime('now')
);

INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
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
    labels:
      {% if cert.letsencrypt.enabled %}
      # 1. HTTPS 主路由
      - traefik.enable=true
      - traefik.http.routers.{{app.code}}.rule=Host(`{{app.code}}.{{ config.domain_suffix }}`)
      - traefik.http.routers.{{app.code}}.entrypoints=websecure
      - traefik.http.routers.{{app.code}}.tls=true
      - traefik.http.routers.{{app.code}}.tls.certresolver=letsencrypt
      - traefik.http.services.{{app.code}}.loadbalancer.server.port=80

      {% else %}
      # 2. HTTP 路由
      - traefik.enable=true
      - traefik.http.routers.{{app.code}}.rule=Host(`{{app.code}}.{{ config.domain_suffix }}`)
      - traefik.http.routers.{{app.code}}.entrypoints=web
      - traefik.http.services.{{app.code}}.loadbalancer.server.port=80
      {% endif %}

networks:
  traefik:
    external: true
',
    datetime('now'),
    datetime('now')
);

INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
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
    datetime('now'),
    datetime('now')
);
