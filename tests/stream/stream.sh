#!/usr/bin/env bash

while IFS='$\n' read -r line; do
  if [ ! -z "$line" ]; then
    V="$(echo $line | jq '.v')"
    echo "{\"input\":$V,\"output\":$(($V*2))}"
  fi
done
