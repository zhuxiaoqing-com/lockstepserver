#!/bin/bash
set -ex

protoc --plugin=protoc-gen-go=protoc-gen-go.exe --go_out=./ ./path/to/your.proto
