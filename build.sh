#!/usr/bin/env bash
##### build.sh — 构建产物输出到 bin/
# 用法:
#   ./build.sh                 # 本机默认（Windows → bin/go-iot.exe，其它 → bin/go-iot）
#   ./build.sh linux           # → bin/go-iot-linux-amd64
#   ./build.sh mac             # 别名 → bin/go-iot-darwin-amd64
#   ./build.sh macm            # 别名 → bin/go-iot-darwin-arm64
#   ./build.sh darwin-amd64    # → bin/go-iot-darwin-amd64
#   ./build.sh darwin-arm64    # → bin/go-iot-darwin-arm64
# 产物按 GOOS-GOARCH 命名，不再使用 go-iot-mac / go-iot-mac_arm

set -euo pipefail

RELEASE="1.1.7"
REPO=$(git config --get remote.origin.url)
GIT_COMMIT=$(git rev-parse --short HEAD)
GO_LD_FLAGS="-s -w -X go-iot/pkg/option.RELEASE=${RELEASE} -X go-iot/pkg/option.COMMIT=${GIT_COMMIT} -X go-iot/pkg/option.REPO=$REPO -X go-iot/pkg/option.BUILD_TIME=$(date "+%Y-%m-%d_%H:%M:%S")"

OUT_DIR="bin"
mkdir -p "${OUT_DIR}"

TARGET="${1:-}"
if [[ -z "${TARGET}" && -n "${2:-}" ]]; then
  TARGET="${2}"
fi

do_build() {
  local goos="$1"
  local goarch="$2"
  local outfile="$3"
  echo "CGO_ENABLED=0 GOOS=${goos} GOARCH=${goarch} go build -v -trimpath -o ${outfile}"
  CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build -v -trimpath -ldflags "${GO_LD_FLAGS}" -o "${outfile}" main.go
  BUILD_FILE_NAME="${outfile}"
}

BUILD_FILE_NAME=""

case "${TARGET}" in
  linux)
    do_build linux amd64 "${OUT_DIR}/go-iot-linux-amd64"
    ;;
  mac|darwin-amd64)
    do_build darwin amd64 "${OUT_DIR}/go-iot-darwin-amd64"
    ;;
  macm|darwin-arm64)
    do_build darwin arm64 "${OUT_DIR}/go-iot-darwin-arm64"
    ;;
  "")
    if [[ "${OS:-}" == Windows* ]]; then
      echo "go build -v -trimpath -o ${OUT_DIR}/go-iot.exe"
      go build -v -trimpath -ldflags "${GO_LD_FLAGS}" -o "${OUT_DIR}/go-iot.exe" main.go
      BUILD_FILE_NAME="${OUT_DIR}/go-iot.exe"
    else
      echo "go build -v -trimpath -o ${OUT_DIR}/go-iot"
      go build -v -trimpath -ldflags "${GO_LD_FLAGS}" -o "${OUT_DIR}/go-iot" main.go
      BUILD_FILE_NAME="${OUT_DIR}/go-iot"
    fi
    ;;
  *)
    echo "unknown target: ${TARGET}" >&2
    echo "usage: $0 [linux|mac|macm|darwin-amd64|darwin-arm64]" >&2
    exit 1
    ;;
esac

echo "build ok: ${BUILD_FILE_NAME}"
