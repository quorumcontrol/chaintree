#!/usr/bin/env bash
set -e -x

pushd "`dirname "$BASH_SOURCE[0]"`/.."
trap popd EXIT

dep ensure

cd vendor/github.com/ry/v8worker2

git clone --depth 1 --branch 6.9.454 https://github.com/v8/v8.git
git clone --depth 1 https://chromium.googlesource.com/chromium/tools/depot_tools.git

./build.py --use_ccache