# Production image for the marketing website (pure static site).
# Build static assets on the host first:
#   ./scripts/build_website_dist.sh
ARG NGINX_BASE=nginx:stable-alpine
FROM ${NGINX_BASE}

COPY dist /usr/share/nginx/html

COPY nginx.conf /etc/nginx/templates/default.conf.template

EXPOSE 80
