#!/usr/bin/env bash

mkdir -p temp && chmod 777 temp

docker run --name debug --env-file .env.local -it -v $(pwd)/temp:/app/database -v $(pwd)/cmd:/app/cmd debug:dev