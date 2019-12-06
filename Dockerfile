FROM golang:stretch as build
WORKDIR /work
ENV XDG_CONFIG_HOME=/work/src/qlik-oss/kustomize-plugins
ENV GO111MODULE=on
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
RUN go get github.com/mikefarah/yq@2.4.1
ENV GO111MODULE=off
RUN go get github.com/hairyhenderson/gomplate/cmd/gomplate
RUN git clone https://github.com/containers/skopeo /go/src/github.com/containers/skopeo
RUN cd /go/src/github.com/containers/skopeo && make binary-local
RUN mv /go/src/github.com/containers/skopeo/skopeo /go/bin/
RUN mv /go/bin/kustomize /go/bin/kustomize.cmd
RUN mv /go/src/qlik-oss/kustomize-plugins/kustomize.wrapper /go/bin/kustomize

FROM debian:stretch
# Note: These .so packages also required for skopeo runtime (as not statically linked)
RUN echo "deb http://deb.debian.org/debian stretch-backports main" >> /etc/apt/sources.list && \
    apt-get update && \
    apt-get install jq libgpgme11-dev libassuan-dev libbtrfs-dev libdevmapper-dev -y && \
    rm -rf /var/lib/apt/lists/*
COPY --from=build /go/bin /usr/local/bin
COPY --from=build /tmp/go/src/qlik-oss/kustomize-plugins/kustomize /root/.config/kustomize
COPY --from=docker.bintray.io/jfrog/jfrog-cli-go:latest /usr/local/bin/jfrog /usr/local/bin/jfrog
