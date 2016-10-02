#!/bin/bash
#Somewhat copied from https://gist.github.com/hailiang/0f22736320abe6be71ce

if [ -z "$1" ]; then
    printf "Usage: $0 -r <Test for race conditions>\n"
    printf "\t\t -c <Generate and open a test coverage report in a browser>\n"
    exit 1
fi

while getopts "rc" FLAG; do
    case $FLAG in
        r)
            RACE=1
            ;;
        c)
            COVER=1
            ;;
    esac
done

cover(){
    echo "mode: count" > profile.out
    for dir in $(find . -type d); do
        if [ -e $dir/*_test.go ]; then
            go test -v -covermode=count -coverprofile=$dir.tmp $dir || exit -1
            cat $dir.tmp | tail -n +2 >> profile.out || exit -1
            rm $dir.tmp
        fi
    done
    go tool cover -html=profile.out
    rm -f profile.out
}

race(){
    for dir in $(find . -type d); do
        if [ -e $dir/*_test.go ]; then
            go test -v -race $dir
        fi
    done

}

if [ "$RACE" == 1 ]; then
    race
fi

if [ "$COVER" == 1 ]; then
    cover
fi
