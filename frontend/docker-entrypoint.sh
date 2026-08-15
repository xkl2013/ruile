#!/bin/sh

APP_MAX_FILE_SIZE_MB=${MAX_FILE_SIZE_MB:-50}
REQUEST_MAX_FILE_SIZE_MB=${UPLOAD_REQUEST_MAX_FILE_SIZE_MB:-100}

case "$APP_MAX_FILE_SIZE_MB" in
  ''|*[!0-9]*) ;;
  *)
    case "$REQUEST_MAX_FILE_SIZE_MB" in
      ''|*[!0-9]*) REQUEST_MAX_FILE_SIZE_MB=$APP_MAX_FILE_SIZE_MB ;;
      *)
        if [ "$APP_MAX_FILE_SIZE_MB" -gt "$REQUEST_MAX_FILE_SIZE_MB" ]; then
          REQUEST_MAX_FILE_SIZE_MB=$APP_MAX_FILE_SIZE_MB
        fi
        ;;
    esac
    ;;
esac

# 生成运行时配置文件，注入环境变量到前端
cat > /usr/share/nginx/html/config.js << EOF
window.__RUNTIME_CONFIG__ = {
  MAX_FILE_SIZE_MB: ${APP_MAX_FILE_SIZE_MB}
};
EOF

# 处理 nginx 配置
export MAX_FILE_SIZE=${REQUEST_MAX_FILE_SIZE_MB}M
export APP_HOST=${APP_HOST:-app}
export APP_PORT=${APP_PORT:-8080}
export APP_SCHEME=${APP_SCHEME:-http}
envsubst '${MAX_FILE_SIZE} ${APP_HOST} ${APP_PORT} ${APP_SCHEME}' < /etc/nginx/templates/default.conf.template > /etc/nginx/conf.d/default.conf

# 启动 nginx
exec nginx -g 'daemon off;'
