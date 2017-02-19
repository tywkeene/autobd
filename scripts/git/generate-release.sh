#!/bin/bash

source VERSION

set -e

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
        sed -i "s/MINOR=.*/MINOR=0/" VERSION
        sed -i "s/PATCH=.*/PATCH=0/" VERSION
    fi
}

function bump_minor(){
    echo "Version is: $MAJOR.$(($MINOR+1)).$PATCH"
    if [[ "yes" == $(ask_yes_or_no "is this what you want?") ]]; then
        sed -i "s/MINOR=.*/MINOR=$(($MINOR + 1))/" VERSION
        sed -i "s/PATCH=.*/PATCH=0/" VERSION
    fi
}

function bump_patch(){
    echo "Version is: $MAJOR.$MINOR.$(($PATCH+1))"
    if [[ "yes" == $(ask_yes_or_no "is this what you want?") ]]; then
        sed -i "s/PATCH=.*/PATCH=$(($PATCH + 1))/" VERSION
    fi
}

function generate_changelog(){
    echo "# $(date +%c) Version: $VERSION" >> CHANGELOG.md
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
            generate_changelog
            ;;
        M) bump_major
            generate_changelog
            ;;
        p) bump_patch
            generate_changelog
            ;;
    esac
done
