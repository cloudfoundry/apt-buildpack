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
  outputDir="${ROOTDIR}/build/ruby-buildpack"

  while [[ "${#}" != 0 ]]; do
    case "${1}" in
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
    util::print::info "Dowloading ruby-buildpack..."
    curl -sJL -o ruby_buildpack_src.zip "https://github.com/cloudfoundry/ruby-buildpack/archive/refs/heads/master.zip"
    unzip -q ruby_buildpack_src.zip
    cd ruby-buildpack-master
    ./scripts/package.sh --version 1.2.3 --stack "${stack}" --cached --output "${outputDir}/ruby-buildpack.zip"
  popd
}

main "${@:-}"
