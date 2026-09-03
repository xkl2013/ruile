# Production image. Build static assets on the host first:
#   ./scripts/build_admin_dist.sh
ARG NGINX_BASE=nginx:stable-alpine
FROM ${NGINX_BASE}

COPY dist /usr/share/nginx/html/admin

COPY nginx.conf /etc/nginx/templates/default.conf.template

COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

ENV MAX_FILE_SIZE_MB=50
ENV UPLOAD_REQUEST_MAX_FILE_SIZE_MB=100

EXPOSE 80

ENTRYPOINT ["/docker-entrypoint.sh"]
