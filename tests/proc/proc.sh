#!/usr/bin/env bash

DATA="$(cat $@)"
echo "{\"input\":\"$DATA\"}" | jq