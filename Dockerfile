FROM golang:stretch as build
WORKDIR /work
ENV XDG_CONFIG_HOME=/work/src/qlik-oss/kustomize-plugins
ENV GO111MODULE=on
RUN apt-get update
RUN apt-get install gcc curl make -y
RUN mkdir -p /go/src/qlik-oss/kustomize-plugins
RUN curl https://get.helm.sh/helm-v2.15.0-linux-amd64.tar.gz | tar xz
RUN cp linux-amd64/helm /go/bin/
COPY . /go/src/qlik-oss/kustomize-plugins/
RUN cd /go/src/qlik-oss/kustomize-plugins && make
RUN find /go/src/qlik-oss/kustomize-plugins -name \*.so -exec cp --parents \{} /tmp \;
RUN go get github.com/mikefarah/yq@2.4.1
ENV GO111MODULE=off
RUN go get github.com/hairyhenderson/gomplate/cmd/gomplate
RUN mv /go/bin/kustomize /go/bin/kustomize.cmd
RUN mv /go/src/qlik-oss/kustomize-plugins/kustomize.wrapper /go/bin/kustomize

FROM debian:stretch

COPY --from=build /go/bin /usr/local/bin
COPY --from=build /tmp/go/src/qlik-oss/kustomize-plugins/kustomize /root/.config/kustomize
