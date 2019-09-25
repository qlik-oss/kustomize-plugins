#!/bin/bash
PROJECT_DIR="$( pwd )"
KUSTOMIZE_DIR="$(pwd)/kustomize"
DEST_DIR=$HOME/.config/kustomize

cd $KUSTOMIZE_DIR
find . -iname '*.go' -exec sh -c 'f="{}"; GO111MODULE=on go build -buildmode plugin -o "echo $f | sed \"s/.go/.so/\"" $f' \;
