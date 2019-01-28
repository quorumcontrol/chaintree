#!/usr/bin/env bash

pushd `dirname $BASH_SOURCE[0]`
trap popd EXIT

set -x -e

rm -f node_modules/bignumber.js/bignumber.mjs

parcel build --no-minify --bundle-node-modules index.js

