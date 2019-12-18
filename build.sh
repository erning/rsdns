#!/bin/bash

if [ "$#" -ge 1 ]; then
  OS=$1
fi

if [ "$OS" == "linux" ]; then
  GOOS=linux
  GOARCH=amd64
elif [ "$OS" == "mac" ]; then
  GOOS=darwin
  GOARCH=amd64
fi

export GOARCH
export GOOS

go build
