-- v0.5.3: Nginx 应用

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
