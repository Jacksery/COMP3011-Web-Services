#!/bin/sh
set -e

# Ensure /work is a directory (host ./data must be a directory)
if [ -e /work ] && [ ! -d /work ]; then
  echo "/work exists and is not a directory; remove or rename the host ./data file and create a directory" >&2
  exit 1
fi

mkdir -p /work

# If DB already exists in data dir, just ensure permissions
if [ -f /work/retailDB.sqlite ]; then
  echo "init-db: found /work/retailDB.sqlite, ensuring ownership/permissions"
  chown 1000:1000 /work/retailDB.sqlite || true
  # Make DB writable by the app regardless of UID mapping in CI runners
  chmod 666 /work/retailDB.sqlite || true

# Else if a DB file exists in repo root (mounted at /src), copy it into /work
elif [ -f /src/retailDB.sqlite ]; then
  echo "init-db: copying /src/retailDB.sqlite -> /work/retailDB.sqlite"
  cp /src/retailDB.sqlite /work/retailDB.sqlite
  chown 1000:1000 /work/retailDB.sqlite || true
  # If the file from the repo is read-only, make it writable for the runtime app
  chmod 666 /work/retailDB.sqlite || true

# Otherwise create an empty DB file
else
  echo "init-db: creating empty /work/retailDB.sqlite"
  touch /work/retailDB.sqlite
  chown 1000:1000 /work/retailDB.sqlite || true
  chmod 666 /work/retailDB.sqlite || true
fi

echo "init-db: done"
