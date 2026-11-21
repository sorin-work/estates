#!/bin/sh
# Restores a dump made by backup.sh into the Docker Compose database:
#
#   scripts/restore.sh backups/estates-20261121T030000Z.dump
#
# The dump goes into a new database first, in one transaction, so a broken
# or incomplete dump changes nothing. Only then is the API stopped and the
# new database swapped in by renaming. The old one is kept as
# estates_before_restore_<time> until you drop it.
set -eu

file=${1:-}
if [ -z "$file" ] || [ ! -f "$file" ]; then
  echo "usage: scripts/restore.sh backups/estates-....dump" >&2
  exit 2
fi

# start_api starts the API again; on a fresh machine it was never created.
start_api() {
  docker compose start api 2>/dev/null || echo "the API is not created yet: run docker compose up -d"
}

db() {
  docker compose exec -T db "$@"
}

psql_admin() {
  db psql -U estates -d postgres -v ON_ERROR_STOP=1 -tA "$@"
}

stamp=$(date -u +%Y%m%dT%H%M%SZ)
old="estates_before_restore_$stamp"

echo "restoring $file into a new database, estates_restore"
psql_admin -c 'DROP DATABASE IF EXISTS estates_restore'
psql_admin -c 'CREATE DATABASE estates_restore'
if ! db pg_restore -U estates -d estates_restore --no-owner --single-transaction --exit-on-error <"$file"; then
  psql_admin -c 'DROP DATABASE IF EXISTS estates_restore'
  echo "the dump could not be restored; the live database was not touched" >&2
  exit 1
fi

echo "stopping the API"
docker compose stop api

# Nothing may still be using the live database, or the rename fails half
# way or a running backup reads the wrong data. Most often this is the
# backup service in the middle of a dump: try again in a minute.
busy=$(psql_admin -c "SELECT count(*) FROM pg_stat_activity WHERE datname = 'estates'")
if [ "$busy" != 0 ]; then
  echo "$busy connection(s) still use the estates database; nothing was swapped" >&2
  start_api
  exit 1
fi

psql_admin <<SQL
BEGIN;
ALTER DATABASE estates RENAME TO $old;
ALTER DATABASE estates_restore RENAME TO estates;
COMMIT;
SQL

echo "starting the API"
start_api

cat <<EOF
Restored. The previous database is kept as $old.
Check the app, then remove it with:
  docker compose exec db dropdb -U estates $old
To undo the restore instead, stop the API and rename it back.
EOF
