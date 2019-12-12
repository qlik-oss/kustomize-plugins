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

RUN git clone https://github.com/ashwathishiva/troubleshoot.git; cd troubleshoot; git checkout custom_text_analyzer; make preflight; ls -ltr bin; mv bin/preflight /go/bin; echo "done moving preflight"; make support-bundle; ls -ltr bin; mv bin/support-bundle /go/bin; echo "done moving support-bundle"; rm -rf troubleshoot

# RUN git clone https://github.com/bearium/troubleshoot.git; cd troubleshoot; git checkout stdout_results; make preflight; ls -ltr bin; mv bin/preflight /go/bin; echo "done moving preflight"
    # rm -rf troubleshoot

# RUN curl -Lo https://github.com/replicatedhq/troubleshoot/releases/download/v0.9.0/preflight_0.19.0_linux_amd64-alpha.tar.gz
# RUN tar xzvf preflight_0.9.0_linux_amd64-alpha.tar.gz
# RUN mv bin/preflight /usr/local/bin

RUN go get github.com/hairyhenderson/gomplate/cmd/gomplate
RUN mv /go/bin/kustomize /go/bin/kustomize.cmd
RUN mv /go/src/qlik-oss/kustomize-plugins/kustomize.wrapper /go/bin/kustomize

FROM debian:stretch

COPY --from=build /go/bin /usr/local/bin
COPY --from=build /tmp/go/src/qlik-oss/kustomize-plugins/kustomize /root/.config/kustomize
