#!/usr/bin/env bash

ROOT=`cd $(dirname $0)/../; pwd`
cd ${ROOT}

source "${ROOT}/scripts/version.sh"
version::get_version_vars

GOLDFLAGS="${GOLDFLAGS=-s -w} $(version::ldflags)"

cat << EOF
GOLDFLAGS="$GOLDFLAGS"
CGO_ENABLED=1
EOF
