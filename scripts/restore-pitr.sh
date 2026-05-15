#!/usr/bin/env sh
set -eu
: "${PGDATA:?PGDATA is required}"
: "${BASE_BACKUP:?BASE_BACKUP is required}"
: "${RECOVERY_TARGET_TIME:?RECOVERY_TARGET_TIME is required}"
rm -rf "$PGDATA"/*
tar -xzf "$BASE_BACKUP" -C "$PGDATA"
printf "restore_command = 'aws s3 cp s3://%s/wal/%%f %%p'
recovery_target_time = '%s'
" "${BACKUP_BUCKET:?BACKUP_BUCKET is required}" "$RECOVERY_TARGET_TIME" >> "$PGDATA/postgresql.auto.conf"
touch "$PGDATA/recovery.signal"
