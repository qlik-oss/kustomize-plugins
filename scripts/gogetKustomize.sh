#!/usr/bin/env bash

kustomizeVersionShort=${1}
kustomizeVersionLong=kustomize/${kustomizeVersionShort}

tagSha=`curl -s https://api.github.com/repos/kubernetes-sigs/kustomize/git/ref/tags/${kustomizeVersionLong} | jq -r '.object.sha'`
gitCommit=`curl -s https://api.github.com/repos/kubernetes-sigs/kustomize/git/tags/${tagSha} | jq -r '.object.sha'`

kustomizeVersionFlag="-X \"sigs.k8s.io/kustomize/kustomize/v3/provenance.version=Qlik build for tag ${kustomizeVersionLong}\""
gitCommitFlag="-X sigs.k8s.io/kustomize/kustomize/v3/provenance.gitCommit=${gitCommit}"
dateFlag="-X sigs.k8s.io/kustomize/kustomize/v3/provenance.buildDate=`date -u +'%Y-%m-%dT%H:%M:%SZ'`"

GO111MODULE=on go get -ldflags "${kustomizeVersionFlag} ${gitCommitFlag} ${dateFlag}" sigs.k8s.io/kustomize/kustomize/v3@${kustomizeVersionShort}
