#!/bin/bash

cd "$(dirname "$0")" || exit 1

find . -type d -name tmp -prune -o -type f -name '*.tf' -exec dirname {} \; | sort -u | xargs -n 1 tflint --enable-rule=terraform_unused_declarations

echo -e "---------------------------------------------------------"
