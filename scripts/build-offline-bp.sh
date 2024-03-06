#!/usr/bin/env bash

set -eu
set -o pipefail

ROOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOTDIR

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${ROOTDIR}/scripts/.util/print.sh"

function main() {
  local stack
  stack="cflinuxfs4"
  outputDir="${ROOTDIR}/build/buildpacks"

  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --buildpack)
        buildpack="${2}"
        shift 2
        ;;

      --stack)
        stack="${2}"
        shift 2
        ;;

      --outputDir)
        outputDir="${2}"
        shift 2
        ;;

      "")
        # skip if the argument is empty
        shift 1
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  mkdir -p "${outputDir}"
  pushd "${outputDir}" > /dev/null
    util::print::info "Dowloading ${buildpack} buildpack ..."
    curl -sJL -o "${buildpack}_buildpack_src.zip" "https://github.com/cloudfoundry/${buildpack}-buildpack/archive/refs/heads/master.zip"
    unzip -q "${buildpack}_buildpack_src.zip"
    cd "${buildpack}-buildpack-master"
    ./scripts/package.sh --version 1.2.3 --stack "${stack}" --cached --output "${outputDir}/${buildpack}-buildpack.zip"
  popd
}

main "${@:-}"
