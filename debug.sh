#!/usr/bin/env bash

docker run --rm --name test --env-file .env.local -v $(pwd)/temp:/app/database test:dev