#!/bin/bash

source VERSION

function ask_yes_or_no() {
    read -p "$1 (y/n): "
    case $(echo $REPLY | tr '[A-Z]' '[a-z]') in
        y|yes) echo "yes" ;;
        *)     echo "no" ;;
    esac
}

function bump_major(){
    echo "Version is: $(($MAJOR+1)).$MINOR.$PATCH"
    if [[ "yes" == $(ask_yes_or_no "is this what you want?") ]]; then
        sed -i "s/MAJOR=.*/MAJOR=$(($MAJOR + 1))/" VERSION
    fi
}

function bump_minor(){
    echo "Version is: $MAJOR.$(($MINOR+1)).$PATCH"
    sed -i "s/MINOR=.*/MINOR=$(($MINOR + 1))/" VERSION
}

function bump_patch(){
    echo "Version is: $MAJOR.$MINOR.$(($PATCH+1))"
    sed -i "s/PATCH=.*/PATCH=$(($PATCH + 1))/" VERSION
}

function usage(){
        printf "Usage: $0 -M [major] -m [minor] -p [patch] -h [print this message]\n"
}

if [ -z "$1" ]; then
    usage
    exit -1
fi

while getopts "hmMp" opt; do
    case "$opt" in
        h) usage
            ;;
        m) bump_minor
            ;;
        M) bump_major
            ;;
        p) bump_patch
            ;;
    esac
done
