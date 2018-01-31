#!/bin/sh
if docker inspect schwift-testing &>/dev/null; then
  echo 'Already running.'
else
  # The `readlink -f` converts the path to repo/testing/data to an absolute path.
  exec docker run --name schwift-testing -P -v "$(readlink -f "$(dirname $0)")/data:/swift/nodes" -t bouncestorage/swift-aio
fi
