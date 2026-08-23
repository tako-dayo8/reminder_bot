#!/usr/bin/env bash

docker image prune -f

docker build -t debug:dev -f "debug.Dockerfile" .