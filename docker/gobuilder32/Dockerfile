FROM i386/debian
WORKDIR /srv/
COPY go-386.tar.gz /srv/
ENV PATH=$PATH:/srv/go/bin/:/srv/gocode/bin GOPATH=/srv/gocode:/srv/go
VOLUME /srv/gocode/bin/ /srv/gocode/src/ /srv/myapp/
RUN apt-get update \
    && apt-get install -y devscripts dh-systemd \
    && tar xf /srv/go-386.tar.gz -C /srv/ \
    && mkdir -p /srv/gocode/bin/ /srv/gocode/src/ /srv/myapp/ \
    && rm -f /srv/go-386.tar.gz \
    && apt-get clean
