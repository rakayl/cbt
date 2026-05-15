#!/usr/bin/env sh
set -eu
: "${DATABASE_URL:?DATABASE_URL is required}"
: "${BACKUP_DIR:=./backups}"
mkdir -p "$BACKUP_DIR"
file="$BACKUP_DIR/cbt-$(date +%Y%m%d%H%M%S).sql.gz"
pg_dump "$DATABASE_URL" | gzip > "$file"
find "$BACKUP_DIR" -type f -name 'cbt-*.sql.gz' -mtime +"${BACKUP_RETENTION_DAYS:-30}" -delete
echo "$file"
