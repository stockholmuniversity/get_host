FROM jenkins-agent/stretch

ENV GOLANG_VERSION 1.9.4

RUN set -eux; \
        goRelArch='linux-amd64'; \
        goRelSha256='15b0937615809f87321a457bb1265f946f9f6e736c563d6c5e0bd2c22e44f779'; \
        url="https://golang.org/dl/go${GOLANG_VERSION}.${goRelArch}.tar.gz"; \

        wget -O go.tgz "$url"; \
        echo "${goRelSha256} *go.tgz" | sha256sum -c -; \
        tar -C /usr/local -xzf go.tgz; \
        rm go.tgz; \
        /usr/local/go/bin/go version

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin"
