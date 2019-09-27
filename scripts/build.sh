#!/usr/bin/env bash

pluginModulesDir="$(pwd)/kustomize/plugin/qlik.com/v1"
for d in ${pluginModulesDir}/*/ ; do
    cd ${d}
    find . -iname '*.go' -exec sh -c 'f="{}"; GO111MODULE=on go build -buildmode plugin -o $(echo $f | sed "s/\.go/.so/") $f' \;
done