#!/usr/bin/env bash
set -euo pipefail

export ROOT="$( dirname "${BASH_SOURCE[0]}" )/.."
if [ ! -f "$ROOT/.bin/ginkgo" ]; then
  (cd "$ROOT/src/apt/vendor/github.com/onsi/ginkgo/ginkgo/" && go install)
fi

GINKGO_NODES=${GINKGO_NODES:-3}
GINKGO_ATTEMPTS=${GINKGO_ATTEMPTS:-1}
export CF_STACK=${CF_STACK:-cflinuxfs3}

cd $ROOT/src/apt/integration
ginkgo -r --flakeAttempts=$GINKGO_ATTEMPTS -nodes $GINKGO_NODES
