#!/usr/bin/env bash

find . -name "*.so" -type f -delete
echo ""

pluginModulesDir="$(pwd)/kustomize/plugin/qlik.com/v1"
for d in ${pluginModulesDir}/*/ ; do
    cd ${d}
    echo "building a plugin in: ${d}"

    find . -iname '*.go' ! -iname '*_test.go' -exec sh -c 'f="{}"; GO111MODULE=on go build -buildmode plugin -o $(echo $f | sed "s/\.go/.so/") $f' \;
    if ls ${d}/*.so 1> /dev/null 2>&1; then
        echo "successfully built a plugin in: ${d}"
        echo ""
    else
        echo "failed to built a plugin in: ${d}"
        exit 1
    fi
done