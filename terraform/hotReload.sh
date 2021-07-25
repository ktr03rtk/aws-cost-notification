#!/bin/bash

cd "$(dirname "$0")" || exit 1

reflex -r '\.tf$' ./exe.sh
