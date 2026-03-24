-- v0.4.5: 更新 Traefik 和 Nginx docker-compose 模板，支持 Let's Encrypt 自动证书
-- 本地开发（letsencrypt.enabled=false）保持 HTTP，生产环境启用后自动切换 HTTPS

-- 更新 Traefik docker-compose.yml.jinja
UPDATE application_config_file
SET content = 'services:
  traefik:
    image: traefik:3
    container_name: traefik
    restart: unless-stopped
    entrypoint: ["sh", "-c", "chmod 600 /etc/traefik/acme.json && exec traefik"]
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"
    networks:
      - traefik
    volumes:
      - {{ app.physical_app_data_dir }}/traefik.yml:/etc/traefik/traefik.yml:ro
      - {{ app.physical_app_data_dir }}/dynamic:/etc/traefik/dynamic:ro
      - {{ app.physical_app_data_dir }}/certs:/etc/traefik/certs:ro
      - {{ app.physical_app_data_dir }}/acme.json:/etc/traefik/acme.json
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      - TRAEFIK_LOG_LEVEL=${TRAEFIK_LOG_LEVEL}
      - TRAEFIK_DASHBOARD=${TRAEFIK_DASHBOARD}
      - TRAEFIK_DASHBOARD_INSECURE=${TRAEFIK_DASHBOARD_INSECURE}
    labels:
      - "traefik.enable=true"
      {% if cert.letsencrypt.enabled %}
      - "traefik.http.routers.traefik-http.rule=Host(`{{ traefik.dashboard_domain }}`)"
      - "traefik.http.routers.traefik-http.entrypoints=web"
      - "traefik.http.routers.traefik-http.middlewares=redirect-to-https@docker"
      - "traefik.http.routers.traefik-dashboard.rule=Host(`{{ traefik.dashboard_domain }}`)"
      - "traefik.http.routers.traefik-dashboard.entrypoints=websecure"
      - "traefik.http.routers.traefik-dashboard.service=api@internal"
      - "traefik.http.routers.traefik-dashboard.tls.certresolver=letsencrypt"
      - "traefik.http.middlewares.redirect-to-https.redirectscheme.scheme=https"
      - "traefik.http.middlewares.redirect-to-https.redirectscheme.permanent=true"
      {% else %}
      - "traefik.http.routers.traefik-dashboard.rule=Host(`{{ traefik.dashboard_domain }}`)"
      - "traefik.http.routers.traefik-dashboard.entrypoints=web"
      - "traefik.http.routers.traefik-dashboard.service=api@internal"
      {% endif %}

networks:
  traefik:
    name: traefik
    driver: bridge
',
    updated_at = datetime('now')
WHERE path = 'docker-compose.yml.jinja'
  AND application_id = (SELECT id FROM application WHERE code = 'traefik');

-- 更新 Nginx docker-compose.yml.jinja
UPDATE application_config_file
SET content = 'services:
  nginx:
    image: nginx:alpine
    container_name: nginx
    restart: unless-stopped
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      {% if cert.letsencrypt.enabled %}
      - "traefik.http.routers.nginx-http.rule=Host(`nginx.{{ config.domain_suffix }}`)"
      - "traefik.http.routers.nginx-http.entrypoints=web"
      - "traefik.http.routers.nginx-http.middlewares=redirect-to-https@docker"
      - "traefik.http.routers.nginx.rule=Host(`nginx.{{ config.domain_suffix }}`)"
      - "traefik.http.routers.nginx.entrypoints=websecure"
      - "traefik.http.routers.nginx.tls.certresolver=letsencrypt"
      - "traefik.http.services.nginx.loadbalancer.server.port=80"
      {% else %}
      - "traefik.http.routers.nginx.rule=Host(`nginx.{{ config.domain_suffix }}`)"
      - "traefik.http.routers.nginx.entrypoints=web"
      - "traefik.http.services.nginx.loadbalancer.server.port=80"
      {% endif %}

networks:
  traefik:
    external: true
',
    updated_at = datetime('now')
WHERE path = 'docker-compose.yml.jinja'
  AND application_id = (SELECT id FROM application WHERE code = 'nginx');
