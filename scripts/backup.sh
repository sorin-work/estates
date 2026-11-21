#!/bin/sh
# Dumps the database every BACKUP_INTERVAL into BACKUP_DIR and keeps the
# newest BACKUP_KEEP scheduled dumps. "backup.sh once" takes one manual
# dump and exits; manual dumps are named differently and never pruned.
#
# Connection settings come from the usual libpq variables (PGHOST, PGUSER,
# PGPASSWORD, PGDATABASE). Docker Compose runs this in the backup service.
set -u

# Dumps hold tenants' personal data: readable by the owner only.
umask 077

BACKUP_DIR=${BACKUP_DIR:-/backups}
BACKUP_INTERVAL=${BACKUP_INTERVAL:-24h}
BACKUP_KEEP=${BACKUP_KEEP:-14}

partial=''

log() {
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) backup: $*"
}

stamp() {
  date -u +%Y%m%dT%H%M%SZ
}

# seconds turns 90, 90s, 15m, 6h or 1d into seconds, or prints nothing
# when the value is not one of those.
seconds() {
  num=${1%[smhd]}
  case "$num" in
    '' | *[!0-9]*) return ;;
  esac
  case "$1" in
    *d) echo $((num * 86400)) ;;
    *h) echo $((num * 3600)) ;;
    *m) echo $((num * 60)) ;;
    *) echo "$num" ;;
  esac
}

# dump writes one dump named $1. It goes to a hidden .partial name first and
# is renamed when pg_dump has finished, so a half-written file never looks
# like a backup.
dump() {
  name=$1
  partial="$BACKUP_DIR/.$name.partial"
  if pg_dump --format=custom --file="$partial"; then
    mv "$partial" "$BACKUP_DIR/$name"
    partial=''
    log "wrote $name ($(du -h "$BACKUP_DIR/$name" | cut -f1))"
  else
    rm -f "$partial"
    partial=''
    log "pg_dump failed, keeping the previous backups"
    return 1
  fi
}

# scheduled lists the scheduled dumps oldest first, leaving out (and
# logging) any dated after now: a clock that ran ahead must not make new
# dumps look older than them.
scheduled() {
  now=$(stamp)
  for f in "$BACKUP_DIR"/estates-[0-9]*.dump; do
    [ -e "$f" ] || continue
    s=$(basename "$f" .dump)
    s=${s#estates-}
    if [ "$s" \> "$now" ]; then
      log "ignoring $(basename "$f"): dated after now, check the clock" >&2
      continue
    fi
    echo "$f"
  done | sort
}

# prune removes all but the newest BACKUP_KEEP scheduled dumps, and never
# the one named $1, just written.
prune() {
  keep=$1
  list=$(scheduled)
  count=$(printf '%s\n' "$list" | grep -c . || true)
  extra=$((count - BACKUP_KEEP))
  [ "$extra" -gt 0 ] || return 0
  printf '%s\n' "$list" | head -n "$extra" | while read -r old; do
    [ "$(basename "$old")" = "$keep" ] && continue
    rm -f "$old"
    log "removed $(basename "$old")"
  done
}

# recent reports whether the newest scheduled dump is less than one
# interval old, so a restart does not take a dump and prune an old one.
recent() {
  newest=$(scheduled | tail -n 1)
  [ -n "$newest" ] || return 1
  since=$(date -u -d "@$(($(date +%s) - interval))" +%Y%m%dT%H%M%SZ)
  s=$(basename "$newest" .dump)
  [ "${s#estates-}" \> "$since" ]
}

stop() {
  [ -n "$partial" ] && rm -f "$partial"
  log "stopping"
  exit 0
}

case "$BACKUP_KEEP" in
  '' | *[!0-9]* | 0)
    log "BACKUP_KEEP must be a whole number above 0, got '$BACKUP_KEEP'"
    exit 2
    ;;
esac
interval=$(seconds "$BACKUP_INTERVAL")
case "$interval" in
  '' | 0)
    log "BACKUP_INTERVAL must be a number of seconds or a number followed by s, m, h or d (30m, 6h, 1d), got '$BACKUP_INTERVAL'"
    exit 2
    ;;
esac

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
# Tighten files written before the umask was set, and clear dumps that were
# cut off by a crash or a stop.
find "$BACKUP_DIR" -maxdepth 1 -type f -name 'estates-*.dump' -exec chmod 600 {} +
find "$BACKUP_DIR" -maxdepth 1 -type f -name '.estates-*.partial' -exec rm -f {} +

trap stop TERM INT

if [ "${1:-}" = once ]; then
  dump "estates-manual-$(stamp).dump"
  exit
fi

log "every $BACKUP_INTERVAL into $BACKUP_DIR, keeping $BACKUP_KEEP"
if recent; then
  log "the last backup is less than $BACKUP_INTERVAL old, waiting for the next one"
  first_wait=1
else
  first_wait=0
fi
while :; do
  if [ "$first_wait" = 0 ]; then
    # A failed run is logged and retried at the next interval, rather than
    # exiting and restarting in a loop.
    name="estates-$(stamp).dump"
    dump "$name" && prune "$name"
  fi
  first_wait=0
  # Sleep in the background so a stop signal is handled straight away.
  sleep "$interval" &
  wait $!
done
