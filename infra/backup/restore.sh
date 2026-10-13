#!/bin/sh
# Restores a database dump and/or an uploads archive from /backups.
#   RESTORE_DB=db-20260101-020000.dump  RESTORE_UPLOADS=uploads-20260101-020000.tgz  CONFIRM=yes
# TARGET_DB defaults to the live database `app`: STOP THE APP FIRST (make down-prod keeps the data).
# Restore into another name (TARGET_DB=app_check) to inspect a backup without touching live data.
set -eu
[ -n "${RESTORE_DB:-}${RESTORE_UPLOADS:-}" ] || { echo "set RESTORE_DB and/or RESTORE_UPLOADS (file names inside ./backups)"; exit 2; }
[ "${CONFIRM:-}" = yes ] || { echo "this overwrites data; re-run with CONFIRM=yes"; exit 2; }

if [ -n "${RESTORE_DB:-}" ]; then
  [ -f "/backups/$RESTORE_DB" ] || { echo "no such file: $RESTORE_DB"; exit 1; }
  psql -d postgres -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS \"$TARGET_DB\" WITH (FORCE)" -c "CREATE DATABASE \"$TARGET_DB\""
  pg_restore --no-owner --exit-on-error -d "$TARGET_DB" "/backups/$RESTORE_DB"
  echo "database restored into $TARGET_DB"
fi
if [ -n "${RESTORE_UPLOADS:-}" ]; then
  [ -f "/backups/$RESTORE_UPLOADS" ] || { echo "no such file: $RESTORE_UPLOADS"; exit 1; }
  tar xzf "/backups/$RESTORE_UPLOADS" -C /data/uploads
  echo "uploads restored"
fi
