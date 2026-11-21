# Backups, restores and upgrades

Everything estates knows lives in one Postgres database, the uploaded
documents included. Backing up that database backs up everything, and
losing it loses everything: leases, the payment history and the signed
PDFs at once.

## Scheduled backups

Docker Compose runs a `backup` service next to the database. It takes a
dump when it starts and then every `BACKUP_INTERVAL`, writes it to
`./backups` (`BACKUP_PATH`) as `estates-<UTC time>.dump`, and keeps the
newest `BACKUP_KEEP` files.

| Variable          | Default     | Notes                                          |
| ----------------- | ----------- | ---------------------------------------------- |
| `BACKUP_PATH`     | `./backups` | Folder on the host the dumps are written to    |
| `BACKUP_INTERVAL` | `24h`       | `30m`, `6h`, `1d` or a number of seconds       |
| `BACKUP_KEEP`     | `14`        | How many dumps to keep; older ones are removed |

Retention counts files, not days, so the disk use is predictable: 14 dumps
a day apart is two weeks, 14 dumps an hour apart is half a day.

Each dump is written under a temporary name and renamed when `pg_dump` has
finished, so a file named `estates-*.dump` is always complete. If a dump
fails (the database is down, the disk is full) the service logs it, keeps
the older dumps and tries again at the next interval. Check on it with
`docker compose logs backup`.

Dumps use the Postgres custom format, which is compressed. Documents are
mostly PDFs and photos, which barely compress, so expect a dump to be about
the size of everything you have uploaded plus a few megabytes.

To take a backup right now:

```bash
make backup    # or: docker compose run --rm backup once
```

The files are created by the container's root user, so on Linux you may
need `sudo` to delete them by hand.

## A copy somewhere else

`./backups` is on the same disk as the database. It protects against
mistakes (a deleted lease, a bad upgrade), not against losing the machine.
Copy the folder somewhere else on a schedule of your own, for example with
cron and `rclone`, `rsync` or a cloud command line:

```bash
rclone copy ./backups remote:estates-backups
aws s3 sync ./backups s3://my-bucket/estates-backups
```

The dumps contain your tenants' names, contact details and ID copies.
Keep the copy somewhere private, and encrypt it if the storage is not your
own.

## Checking a backup

List what a dump contains without restoring it:

```bash
docker compose exec -T db pg_restore --list < backups/estates-20261121T030000Z.dump | head
```

A dump you have never restored is a hope, not a backup. Now and then,
restore one into a scratch database and look at it:

```bash
docker compose exec -T db createdb -U estates restore_check
docker compose exec -T db pg_restore -U estates -d restore_check --no-owner < backups/estates-20261121T030000Z.dump
docker compose exec -T db psql -U estates -d restore_check -c 'SELECT count(*) FROM leases'
docker compose exec -T db dropdb -U estates restore_check
```

CI does the same on every change: it takes a backup of the demo data and
restores it into a second database.

## Restoring

Restoring replaces everything in the database with the contents of the
dump. Anything entered after the dump was taken is lost, so take a fresh
backup first if the current data is worth anything.

```bash
make restore FILE=backups/estates-20261121T030000Z.dump
```

That stops the API, runs `pg_restore --clean --if-exists` into the
`estates` database and starts the API again. Signed in users stay signed
in, as long as `JWT_SECRET` has not changed.

On a new machine, or after `docker compose down -v`, start the database on
its own first, restore, then start the rest:

```bash
docker compose up -d db
docker compose exec -T db pg_restore -U estates -d estates --no-owner < backups/estates-20261121T030000Z.dump
docker compose up -d
```

Docker Compose sets `SEED_DEMO=true`, so if the backup has no demo account
the API adds one on its next start. Set `SEED_DEMO=false` in `.env` when the
data is your own.

## Upgrading

The API applies its database migrations when it starts. To upgrade:

```bash
make backup
git pull
docker compose up --build -d
docker compose logs api | grep migration
```

The log lists each migration it applied. Migrations are only tested going
forward. To go back to an older version, check it out and restore the
backup you took before upgrading. Do not run the older version against the
upgraded database.
