#!/usr/bin/env bash

ROOT=`cd $(dirname $0)/../; pwd`
cd ${ROOT}

source "${ROOT}/scripts/version.sh"
version::get_version_vars

echo "-s -w $(version::ldflags)"
