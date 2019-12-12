FROM docker:19.03.5 AS docker
FROM golang:1.13.4-stretch as build
WORKDIR /work
ENV XDG_CONFIG_HOME=/work/src/qlik-oss/kustomize-plugins
RUN echo "deb http://deb.debian.org/debian stretch-backports main" >> /etc/apt/sources.list
RUN apt-get update
RUN apt-get install gcc curl make -y 
RUN apt-get install libgpgme11-dev libassuan-dev libbtrfs-dev libdevmapper-dev -y
RUN mkdir -p /go/src/qlik-oss/kustomize-plugins
RUN curl https://get.helm.sh/helm-v2.15.0-linux-amd64.tar.gz | tar xz
RUN cp linux-amd64/helm /go/bin/
COPY . /go/src/qlik-oss/kustomize-plugins/
RUN cd /go/src/qlik-oss/kustomize-plugins && make
RUN find /go/src/qlik-oss/kustomize-plugins -name \*.so -exec cp --parents \{} /tmp \;
RUN GO111MODULE=on go get github.com/mikefarah/yq/v2
RUN go get github.com/hairyhenderson/gomplate/cmd/gomplate
RUN mv /go/bin/kustomize /go/bin/kustomize.cmd
RUN mv /go/src/qlik-oss/kustomize-plugins/kustomize.wrapper /go/bin/kustomize

FROM debian:stretch
RUN apt-get update && apt-get install jq curl -y && rm -rf /var/lib/apt/lists/*
ENV JFROG_CLI_OFFER_CONFIG=false
RUN curl -fL https://getcli.jfrog.io | sh &&\
    mv jfrog /bin
COPY --from=build /go/bin /usr/local/bin
COPY --from=build /tmp/go/src/qlik-oss/kustomize-plugins/kustomize /root/.config/kustomize
COPY --from=docker /usr/local/bin/docker /bin/docker
