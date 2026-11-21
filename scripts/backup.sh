#!/bin/sh
# Dumps the database every BACKUP_INTERVAL into BACKUP_DIR and keeps the
# newest BACKUP_KEEP dumps. "backup.sh once" takes one dump and exits.
#
# Connection settings come from the usual libpq variables (PGHOST, PGUSER,
# PGPASSWORD, PGDATABASE). Docker Compose runs this in the backup service.
set -u

BACKUP_DIR=${BACKUP_DIR:-/backups}
BACKUP_INTERVAL=${BACKUP_INTERVAL:-24h}
BACKUP_KEEP=${BACKUP_KEEP:-14}

log() {
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) backup: $*"
}

# seconds turns 90, 90s, 15m, 6h or 1d into seconds.
seconds() {
  case "$1" in
    *d) echo $((${1%d} * 86400)) ;;
    *h) echo $((${1%h} * 3600)) ;;
    *m) echo $((${1%m} * 60)) ;;
    *s) echo "${1%s}" ;;
    *) echo "$1" ;;
  esac
}

# dump writes one dump. It goes to a temporary name first and is renamed
# when pg_dump has finished, so a half-written file never looks like a
# backup.
dump() {
  name="estates-$(date -u +%Y%m%dT%H%M%SZ).dump"
  tmp="$BACKUP_DIR/.$name.partial"
  if pg_dump --format=custom --file="$tmp"; then
    mv "$tmp" "$BACKUP_DIR/$name"
    log "wrote $name ($(du -h "$BACKUP_DIR/$name" | cut -f1))"
  else
    rm -f "$tmp"
    log "pg_dump failed, keeping the previous backups"
    return 1
  fi
}

# prune removes all but the newest BACKUP_KEEP dumps. The names sort by
# time, so the oldest come first.
prune() {
  count=$(ls "$BACKUP_DIR"/estates-*.dump 2>/dev/null | wc -l)
  extra=$((count - BACKUP_KEEP))
  if [ "$extra" -gt 0 ]; then
    ls "$BACKUP_DIR"/estates-*.dump | sort | head -n "$extra" | while read -r old; do
      rm -f "$old"
      log "removed $(basename "$old")"
    done
  fi
}

case "$BACKUP_KEEP" in
  '' | *[!0-9]* | 0)
    log "BACKUP_KEEP must be a whole number above 0, got '$BACKUP_KEEP'"
    exit 2
    ;;
esac
interval=$(seconds "$BACKUP_INTERVAL")
case "$interval" in
  '' | *[!0-9]* | 0)
    log "BACKUP_INTERVAL must be like 30m, 6h or 1d, got '$BACKUP_INTERVAL'"
    exit 2
    ;;
esac
mkdir -p "$BACKUP_DIR"

if [ "${1:-}" = once ]; then
  dump && prune
  exit
fi

log "every $BACKUP_INTERVAL into $BACKUP_DIR, keeping $BACKUP_KEEP"
while :; do
  # A failed run is logged and retried at the next interval, rather than
  # exiting and restarting in a loop.
  dump && prune
  sleep "$interval"
done
