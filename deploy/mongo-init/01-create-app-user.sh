#!/bin/bash
set -euo pipefail
: "${IRON_DB_NAME:?IRON_DB_NAME required}"
: "${IRON_DB_USER:?IRON_DB_USER required}"
: "${IRON_DB_PASSWORD:?IRON_DB_PASSWORD required}"

mongosh --quiet \
  --username "$MONGO_INITDB_ROOT_USERNAME" \
  --password "$MONGO_INITDB_ROOT_PASSWORD" \
  --authenticationDatabase admin \
  --eval '
    const dbName = process.env.IRON_DB_NAME;
    const appUser = process.env.IRON_DB_USER;
    const appPassword = process.env.IRON_DB_PASSWORD;
    db.getSiblingDB(dbName).createUser({
      user: appUser,
      pwd: appPassword,
      roles: [{ role: "readWrite", db: dbName }]
    });
  '
