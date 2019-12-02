#!/usr/bin/env bash

GOPRIVATE=github.com/quorumcontrol GOOS=js GOARCH=wasm go build -o browsertest/main.wasm