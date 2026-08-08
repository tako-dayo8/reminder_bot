#!/usr/bin/env bash

docker run --rm --name debug --env-file .env.local -v $(pwd)/temp:/app/database debug:dev