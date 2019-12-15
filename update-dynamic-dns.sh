#!/bin/bash

SHA=`command -v sha1sum || command -v shasum`
CURL=`command -v curl`

function usage {
    echo "Usage: $0 {host} {shared_key}"
}

if test -z $1 || test -z $2
then
    usage
    exit 1
fi

HOST=$1
SHARED_KEY=$2
ENDPOINT="http://127.0.0.1:8080/api/plain.php"

TIME=`date +%s`
SIGN=`echo -n "${HOST}${TIME}${SHARED_KEY}" | ${SHA} -t | awk '{print $1}'`
URL="${ENDPOINT}?host=${HOST}&time=${TIME}&sign=${SIGN}&ip=$3"
${CURL} -s "$URL"

