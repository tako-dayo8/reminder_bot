#!/usr/bin/env bash

mkdir -p temp && chmod 777 temp

docker run --rm --name debug --env-file .env.local -v $(pwd)/temp:/app/database debug:dev

trap 'echo "container stopping" & docker container rm -f debug > /dev/null' EXIT