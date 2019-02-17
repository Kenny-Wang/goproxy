### Makefile --- 

## Author: shell909090@gmail.com
## Version: $Id: Makefile,v 0.0 2012/11/02 06:18:14 shell Exp $
## Keywords: 
## X-URL: 
LEVEL=NOTICE

all: clean build

clean:
	rm -rf bin pkg gopath debuild

build:
	mkdir -p gopath/src/github.com/shell909090/
	ln -s "$$PWD" gopath/src/github.com/shell909090/goproxy
	mkdir -p bin
	GOPATH="$$PWD/gopath":"$$GOPATH" go build -o bin/goproxy github.com/shell909090/goproxy/goproxy
	rm -rf gopath

test:
	go test github.com/shell909090/goproxy/tunnel
	go test github.com/shell909090/goproxy/dns
	go test github.com/shell909090/goproxy/ipfilter
	go test github.com/shell909090/goproxy/goproxy

install: build
	install -d $(DESTDIR)/usr/bin/
	install -m 755 -s bin/goproxy $(DESTDIR)/usr/bin/
	install -d $(DESTDIR)/usr/share/goproxy/
	install -m 644 debian/routes.list.gz $(DESTDIR)/usr/share/goproxy/
	install -d $(DESTDIR)/etc/goproxy/
	install -m 644 debian/config.json $(DESTDIR)/etc/goproxy/

build-tar: build
	strip bin/goproxy
	tar cJf ../goproxy-`uname -m`.tar.xz bin/goproxy debian/config.json debian/routes.list.gz

build-deb:
	dpkg-buildpackage
	mkdir -p debuild
	mv -f ../goproxy_* debuild

# compile project with gobuilder and push them to docker.io
build-docker:
	sudo docker run --rm -v "$PWD":/srv/gocode/src/github.com/shell909090/goproxy -w /srv/gocode/src/github.com/shell909090/goproxy gobuilder make build
	docker/goproxy/build.sh
	sudo make clean
	sudo docker tag goproxy shell909090/goproxy
	sudo docker push shell909090/goproxy
	sudo docker run --rm -v "$PWD":/srv/gocode/src/github.com/shell909090/goproxy -w /srv/gocode/src/github.com/shell909090/goproxy gobuilder32 make build
	docker/goproxy32/build.sh
	sudo make clean
	sudo docker tag goproxy32 shell909090/goproxy32
	sudo docker push shell909090/goproxy32

### Makefile ends here
