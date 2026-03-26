-- v0.5.4: Pomelo Orbit 自身应用

-- 插入 Pomelo Orbit 应用
INSERT INTO application (id, name, code, image_pull_policy, status, enabled, created_at, updated_at)
VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KW1',
    'Pomelo Orbit',
    'pomelo-orbit',
    'missing',
    'stopped',
    1,
    datetime('now'),
    datetime('now')
);

-- Pomelo Orbit docker-compose.yml
INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KW2',
    '01KKX2YNPF6VJ9N7QYCWG61KW1',
    'docker-compose.yml.jinja',
    'services:
  pomelo-orbit:
    image: pomelo-orbit:latest
    container_name: pomelo-orbit
    restart: unless-stopped
    networks:
      - traefik
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - {{ app.physical_dir }}/data:/app/data
      - {{ app.physical_dir }}/.env:/app/.env
    ports:
      - "9003:80"
    # environment:
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

-- Pomelo Orbit init.sh
INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KW4',
    '01KKX2YNPF6VJ9N7QYCWG61KW1',
    'init.sh',
    '#!/bin/bash
set -e

# 确保 traefik 网络存在（Traefik 未部署时也能正常启动）
docker network inspect traefik || docker network create traefik
',
    datetime('now'),
    datetime('now')
);

-- Pomelo Orbit .env
INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KW3',
    '01KKX2YNPF6VJ9N7QYCWG61KW1',
    '.env',
    '# JWT 密钥, Fernet 格式, 生成方法
# uv run --project backend python -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())"
POMELO_ORBIT_JWT__SECRET_KEY=00000000000000000000000000000000000000000000

',
    datetime('now'),
    datetime('now')
);
