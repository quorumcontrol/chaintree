#!/usr/bin/env bash

GOOS=js GOARCH=wasm go build -o browsertest/main.wasm