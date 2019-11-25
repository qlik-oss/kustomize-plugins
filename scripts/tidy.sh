#!/usr/bin/env bash

pluginModulesDir="$(pwd)/kustomize/plugin/qlik.com/v1"
for d in ${pluginModulesDir}/*/ ; do
    pushd ${d}
    go get -u ./...
    go mod tidy
    popd
done

pushd "$(pwd)/kustomize/utils"
go get -u ./...
go mod tidy
popd