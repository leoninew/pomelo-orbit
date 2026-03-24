-- v0.4.3: Pomelo Orbit 自身应用

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
      - {{ app.physical_data_dir }}:/app/data
    ports:
      - "9003:80"
    environment:
      - POMELO_ORBIT_JWT__SECRET_KEY=${POMELO_ORBIT_JWT__SECRET_KEY}
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.pomelo-orbit.rule=Host(`pomelo-orbit.{{ config.domain_suffix }}`)"
      - "traefik.http.routers.pomelo-orbit.entrypoints=web"
      - "traefik.http.routers.pomelo-orbit.service=pomelo-orbit"
      - "traefik.http.services.pomelo-orbit.loadbalancer.server.port=80"

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
    '# JWT 密钥（同时用于凭据加密，必须使用 Fernet 格式）
# 生成方法: uv run --project backend python -c "from cryptography.fernet import Fernet; print(Fernet.generate_key().decode())"
POMELO_ORBIT_JWT__SECRET_KEY=your-fernet-key-here

',
    datetime('now'),
    datetime('now')
);
