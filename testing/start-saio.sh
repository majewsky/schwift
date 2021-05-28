#!/bin/bash
set -euo pipefail

if docker inspect schwift-testing &>/dev/null; then
  echo 'Already running.'
else
  exec docker run --name schwift-testing -P -t dockerswiftaio/docker-swift:2.27.0
fi
