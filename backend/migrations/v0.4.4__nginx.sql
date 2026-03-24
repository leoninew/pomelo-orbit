-- v0.4.4: Nginx 应用

-- 插入 nginx 应用
INSERT INTO application (id, name, code, image_pull_policy, status, enabled, created_at, updated_at)
VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KW5',
    'Nginx',
    'nginx',
    'missing',
    'stopped',
    1,
    datetime('now'),
    datetime('now')
);

-- nginx docker-compose.yml.jinja
INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
VALUES (
    '01KKX2YNPF6VJ9N7QYCWG61KW6',
    '01KKX2YNPF6VJ9N7QYCWG61KW5',
    'docker-compose.yml.jinja',
    'services:
  nginx:
    image: nginx:alpine
    container_name: nginx
    restart: unless-stopped
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.nginx.rule=Host(`nginx.{{ config.domain_suffix }}`)"
      - "traefik.http.routers.nginx.entrypoints=web"
      - "traefik.http.services.nginx.loadbalancer.server.port=80"

networks:
  traefik:
    external: true
',
    datetime('now'),
    datetime('now')
);
