#!/usr/bin/env bash

versionOfKustomizeCLI=${1}

kustomizeVersionFlag="-X \"sigs.k8s.io/kustomize/kustomize/v3/provenance.version=Qlik built CLI ${versionOfKustomizeCLI}\""
gitCommitFlag="-X sigs.k8s.io/kustomize/kustomize/v3/provenance.gitCommit=unknown"
dateFlag="-X sigs.k8s.io/kustomize/kustomize/v3/provenance.buildDate=`date -u +'%Y-%m-%dT%H:%M:%SZ'`"

GO111MODULE=on go get -ldflags "${kustomizeVersionFlag} ${gitCommitFlag} ${dateFlag}" sigs.k8s.io/kustomize/kustomize/v3@${versionOfKustomizeCLI}
