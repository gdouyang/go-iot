#!/usr/bin/env bash
##### build.sh

RELEASE=1.0
REPO=$(git config --get remote.origin.url)
GIT_COMMIT=$(git rev-parse --short HEAD)
# Build Flags
GO_LD_FLAGS="-s -w -X go-iot/pkg/option.RELEASE=${RELEASE} -X go-iot/pkg/option.COMMIT=${GIT_COMMIT} -X go-iot/pkg/option.REPO=$REPO -X go-iot/pkg/option.BUILD_TIME=$(date "+%Y-%m-%d_%H:%M:%S")"
# echo $GO_LD_FLAGS
BUILD_FILE_NAME="go-iot"
if [[ "$OS" == Windows* ]];then
  if [[ "$1" == linux || "$2" == linux ]];then
    echo "cross build linux"
    echo CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -trimpath -ldflags \"${GO_LD_FLAGS}\" -o go-iot main.go
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -trimpath -ldflags "${GO_LD_FLAGS}" -o go-iot main.go
  elif [[ "$1" == mac || "$2" == mac ]];then
    echo CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -v -trimpath -ldflags "${GO_LD_FLAGS}" -o go-iot main.go
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -v -trimpath -ldflags "${GO_LD_FLAGS}" -o go-iot main.go
  else
    echo go build -v -trimpath -ldflags \"${GO_LD_FLAGS}\" -o go-iot.exe main.go
    go build -v -trimpath -ldflags "${GO_LD_FLAGS}" -o go-iot.exe main.go
    BUILD_FILE_NAME="go-iot.exe"
  fi
else
echo go build -v -trimpath -ldflags \"${GO_LD_FLAGS}\" -o go-iot main.go
go build -v -trimpath -ldflags "${GO_LD_FLAGS}" -o go-iot main.go
fi
