# Backups, restores and upgrades

Everything estates knows lives in one Postgres database, the uploaded
documents included. Backing up that database backs up everything, and
losing it loses everything: leases, the payment history and the signed
PDFs at once.

## Scheduled backups

Docker Compose runs a `backup` service next to the database. It takes a
dump when it starts, unless the newest one is less than an interval old,
and then every `BACKUP_INTERVAL`. It writes to `./backups` (`BACKUP_PATH`)
as `estates-<UTC time>.dump` and keeps the newest `BACKUP_KEEP` of those.

| Variable          | Default     | Notes                                          |
| ----------------- | ----------- | ---------------------------------------------- |
| `BACKUP_PATH`     | `./backups` | Folder on the host the dumps are written to    |
| `BACKUP_INTERVAL` | `24h`       | Seconds, or a number with `s`, `m`, `h` or `d` |
| `BACKUP_KEEP`     | `14`        | How many scheduled dumps to keep               |

Retention counts files, not days, so the disk use is predictable: 14 dumps
a day apart is two weeks, 14 dumps an hour apart is half a day. The dump
just written is never removed. A dump dated after the current time (the
clock ran ahead at some point) is left alone and logged, so it cannot push
newer dumps out.

Each dump is written under a hidden `.partial` name and renamed when
`pg_dump` has finished, so a file named `estates-*.dump` is always
complete. Partial files left by a crash are removed when the service
starts, and stopping the service removes the one in progress. If a dump
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

Manual dumps are named `estates-manual-<UTC time>.dump`. They do not count
towards `BACKUP_KEEP` and are never removed automatically, so delete them
yourself when you no longer need them.

The folder and the dumps are readable by their owner only, and that owner
is the container's root user, so on Linux you need `sudo` to read, copy or
delete them on the host.

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
sudo cat backups/estates-20261121T030000Z.dump | docker compose exec -T db pg_restore --list | head
```

A dump you have never restored is a hope, not a backup. Now and then,
restore one into a scratch database and look at it:

```bash
docker compose exec -T db createdb -U estates restore_check
sudo cat backups/estates-20261121T030000Z.dump | docker compose exec -T db pg_restore -U estates -d restore_check --no-owner
docker compose exec -T db psql -U estates -d restore_check -c 'SELECT count(*) FROM leases'
docker compose exec -T db dropdb -U estates restore_check
```

CI goes further on every change: it takes a manual backup of the demo
data, adds a table the backup does not have, runs `scripts/restore.sh`, and
checks that the table is gone, the users are back, the old database was
kept and the API signs in again.

## Restoring

```bash
sudo make restore FILE=backups/estates-20261121T030000Z.dump
```

(`sudo` because only root can read the dumps.) That runs `scripts/restore.sh`, which:

1. restores the dump into a new database, `estates_restore`, in a single
   transaction that stops at the first error. A broken or incomplete dump
   leaves the live database untouched;
2. stops the API;
3. refuses to go on, and starts the API again, if anything is still
   connected to the live database (most often the backup service in the
   middle of a dump: wait for it, or `docker compose stop backup`);
4. renames the live database to `estates_before_restore_<UTC time>` and
   `estates_restore` to `estates`, in one transaction;
5. starts the API again.

Anything entered after the dump was taken is in the old database only.
Check the app, then drop the old database with the command the script
prints. To undo the restore, stop the API and rename the two back. Signed
in users stay signed in, as long as `JWT_SECRET` has not changed.

On a new machine, or after `docker compose down -v`, start the database on
its own first and wait for it, restore, then start the rest:

```bash
docker compose up -d --wait db
sudo scripts/restore.sh backups/estates-20261121T030000Z.dump
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
forward. To go back to an older version, check it out, rebuild, and restore
the backup you took before upgrading with `make restore`. The restore goes
into a fresh database, so tables the newer version added do not get in the
way. Do not run the older version against the upgraded database.
