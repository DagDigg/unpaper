#!/bin/bash

rm -rf ./core/db/migrations/*

kubectl schemahero fixtures \
  --input-dir ./artifacts/db/tables \
  --output-dir ./core/db/migrations \
  --dbname unpaper \
  --driver postgres

# Rename fixtures as schemahero doesn't support custom file name,
# and golang-migrate doesn't like different namings
mv ./core/db/migrations/fixtures.sql ./core/db/migrations/01_fixtures.up.sql 

dirs=("users" "customers" "lists" "comments" "posts" "notifications" "follows" "mixes")
for d in "${dirs[@]}"; do
  cd ./backend/$d
  sqlc generate
  cd ../..
done