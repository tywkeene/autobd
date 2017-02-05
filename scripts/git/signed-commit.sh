#!/bin/bash

# Sign and commit. Won't work if you dont have gpg/git/github set up
# to do this. See: https://help.github.com/articles/signing-commits-using-gpg/

if [ -z "$1" ]; then
    echo "Refusing to commit without a message"
    exit 1
fi

git commit -S -m "$1"
