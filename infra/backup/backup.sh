#!/bin/sh
# Dumps the database and archives the uploads into /backups, then prunes old backups.
# Both files are written under a temporary name and renamed only when complete and verified, so a
# crashed run never leaves something that looks like a good backup.
set -eu
ts=$(date +%Y%m%d-%H%M%S)
keep=${BACKUP_KEEP_DAYS:-14}
cd /backups

pg_dump -Fc --no-owner "$PGDATABASE" > ".db-$ts.dump.tmp"
pg_restore -l ".db-$ts.dump.tmp" > /dev/null   # the archive must be readable
mv ".db-$ts.dump.tmp" "db-$ts.dump"

tar czf ".uploads-$ts.tgz.tmp" -C /data/uploads .
tar tzf ".uploads-$ts.tgz.tmp" > /dev/null
mv ".uploads-$ts.tgz.tmp" "uploads-$ts.tgz"

find . -maxdepth 1 \( -name 'db-*.dump' -o -name 'uploads-*.tgz' \) -mtime +"$keep" -delete
echo "backup $ts done: db-$ts.dump ($(du -h "db-$ts.dump" | cut -f1)), uploads-$ts.tgz ($(du -h "uploads-$ts.tgz" | cut -f1))"
