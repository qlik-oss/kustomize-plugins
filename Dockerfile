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
ENV GO111MODULE=off
RUN go get github.com/hairyhenderson/gomplate/cmd/gomplate
RUN go get gopkg.in/mikefarah/yq.v2
RUN mv /go/bin/yq.v2 /go/bin/yq

# ENV KUBECTL_VER 1.15.0
# RUN curl -Lo /go/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VER}/bin/linux/amd64/kubectl \
#     && chmod +x /go/bin/kubectl
RUN curl -Lo kubeval-linux-amd64.tar.gz https://github.com/instrumenta/kubeval/releases/latest/download/kubeval-linux-amd64.tar.gz
RUN tar xf kubeval-linux-amd64.tar.gz
RUN cp kubeval /go/bin
RUN mkdir /root/.config
RUN cp -r /tmp/go/src/qlik-oss/kustomize-plugins/kustomize /root/.config/kustomize
# FROM debian:stretch

# COPY --from=build /go/bin /usr/local/bin
# COPY --from=build /tmp/go/src/qlik-oss/kustomize-plugins/kustomize /root/.config/kustomize