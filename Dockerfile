FROM debian:stretch as build
ENV GOPATH=/work
ENV PATH=$PATH:/usr/local/go/bin:/work/bin
ENV XDG_CONFIG_HOME=/work/src/qlik-oss/kustomize-plugins
ENV GO111MODULE=on
RUN apt-get update
RUN apt-get install gcc curl git make gnupg apt-transport-https lsb-release software-properties-common -y 
RUN curl https://dl.google.com/go/go1.13.3.linux-amd64.tar.gz | tar xz -C /usr/local/
RUN mkdir -p /work/bin && mkdir -p /work/src/qlik-oss/kustomize-plugins
RUN curl https://get.helm.sh/helm-v2.15.0-linux-amd64.tar.gz | tar xz
RUN cp linux-amd64/helm /work/bin/
COPY . /work/src/qlik-oss/kustomize-plugins/
RUN cd /work/src/qlik-oss/kustomize-plugins && make
RUN find /work/src/qlik-oss/kustomize-plugins -name \*.so -exec cp --parents \{} /tmp \;
ENV GO111MODULE=off
RUN go get github.com/hairyhenderson/gomplate/cmd/gomplate
RUN go get gopkg.in/mikefarah/yq.v2
RUN mv /work/bin/yq.v2 /work/bin/yq

FROM debian:stretch

COPY --from=build /work/bin /usr/local/bin
COPY --from=build /tmp/work/src/qlik-oss/kustomize-plugins/kustomize /root/.config/kustomize
