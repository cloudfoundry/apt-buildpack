#!/usr/bin/env bash

set -e
set -u
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${ROOTDIR}/scripts/.util/tools.sh"

function main() {
  local src stack harness
  src="$(find "${ROOTDIR}/src" -mindepth 1 -maxdepth 1 -type d )"
  stack="$(jq -r -S .stack "${ROOTDIR}/config.json")"
  harness="$(jq -r -S .integration.harness "${ROOTDIR}/config.json")"

  util::tools::ginkgo::install --directory "${ROOTDIR}/.bin"
  util::tools::buildpack-packager::install --directory "${ROOTDIR}/.bin"
  util::tools::cf::install --directory "${ROOTDIR}/.bin"

  local cached serial
  cached=true
  serial=true

  if [[ "${src}" == *python ]]; then
    specs::run "${harness}" "uncached" "parallel"
    specs::run "${harness}" "uncached" "serial"

    specs::run "${harness}" "cached" "parallel"
    specs::run "${harness}" "cached" "serial"
  else
    specs::run "${harness}" "uncached" "parallel"
    specs::run "${harness}" "cached" "parallel"
  fi
}

function specs::run() {
  local harness cached serial
  harness="${1}"
  cached="${2}"
  serial="${3}"

  if [[ "${harness}" == "gotest" ]]; then
    specs::gotest::run "${cached}" "${serial}"
  else
    specs::ginkgo::run "${cached}" "${serial}"
  fi
}

function specs::gotest::run() {
  local cached serial nodes
  cached="false"
  serial=""
  nodes=3

  echo "Run ${1} Buildpack"

  if [[ "${1}" == "cached" ]] ; then
    cached="true"
  fi

  if [[ "${2}" == "serial" ]]; then
    nodes=1
    serial="-serial=true"
  fi

  CF_STACK="${CF_STACK:-"${stack}"}" \
  BUILDPACK_FILE="${UNCACHED_BUILDPACK_FILE:-}" \
  GOMAXPROCS="${GOMAXPROCS:-"${nodes}"}" \
    go test \
      -count=1 \
      -timeout=0 \
      -mod vendor \
      -v \
        "${src}/integration" \
          --cached="${cached}" "${serial}"
}

function specs::ginkgo::run(){
  local cached serial nodes
  cached="false"
  serial=""
  nodes="${GINKGO_NODES:-3}"

  echo "Run ${1} Buildpack"

  if [[ "${1}" == "cached" ]] ; then
    cached="true"
  fi

  if [[ "${2}" == "serial" ]]; then
    nodes=1
    serial="-serial=true"
  fi

  CF_STACK="${CF_STACK:-"${stack}"}" \
  BUILDPACK_FILE="${UNCACHED_BUILDPACK_FILE:-}" \
    ginkgo \
      -r \
      -mod vendor \
      --flakeAttempts "${GINKGO_ATTEMPTS:-2}" \
      -nodes ${nodes} \
      --slowSpecThreshold 60 \
        "${src}/integration" \
      -- --cached="${cached}" "${serial}"
}

main "${@:-}"
