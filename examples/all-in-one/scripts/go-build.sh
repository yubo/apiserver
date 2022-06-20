#!/bin/bash

KUBE_ROOT=`cd $(dirname $0)/../; pwd`
cd ${KUBE_ROOT}

OUTFILE=${1:-all-in-one}


source "${KUBE_ROOT}/scripts/version.sh"
kube::version::get_version_vars

goldflags="${GOLDFLAGS=-s -w} $(kube::version::ldflags)"
goasmflags="-trimpath=${KUBE_ROOT}"
gogcflags="${GOGCFLAGS:-} -trimpath=${KUBE_ROOT}"

build_args=(
  -installsuffix static
  -gcflags "${gogcflags:-}"
  -asmflags "${goasmflags:-}"
  -ldflags "${goldflags:-}"
)

set -x
CGO_ENABLED=${CGO_ENABLED:-0} go build "${build_args[@]}" -o ${OUTFILE} ./cmd/all-in-one
