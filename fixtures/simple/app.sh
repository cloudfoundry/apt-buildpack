#!/usr/bin/env bash

tmppipe=$(mktemp -u)
mkfifo "${tmppipe}"
trap 'rm "${tmppipe}"' EXIT

function handler {
  header="HTTP/1.1 200 OK\r\nContent-Type: text/html\r\n\r\n"
  while read -r line; do
    if [ -z "$(echo "$line" | tr -d '\r\n')" ]; then
      break
    fi

    case "${line}" in
      GET[[:space:]]/jq[[:space:]]*)
        echo -e "${header}Jq: $(jq --version)" > "${tmppipe}"
        return
        ;;
      GET[[:space:]]/cf[[:space:]]*)
        echo -e "${header}cf: $(cf --version)" > "${tmppipe}"
        return
        ;;
      GET[[:space:]]/bosh[[:space:]]*)
        echo -e "${header}BOSH: $(bosh2 -v)" > "${tmppipe}"
        return
        ;;
    esac
  done
  echo -e "HTTP/1.1 404 NotFound\r\n\r\n\r\nNot Found" > "${tmppipe}"
}

while true; do
  tail -f "${tmppipe}" | nc -lvN "${PORT:-8080}" | handler;
done
