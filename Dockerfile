FROM debian:latest

WORKDIR /workdir

RUN apt-get update && \
	apt-get install -y git python3 curl cmake unzip autoconf automake libtool perl patch pkg-config ccache g++ build-essential

# Build emscripten
RUN git clone https://github.com/EasyRPG/buildscripts && \
    cd buildscripts && \
    cd emscripten && \
    ./0_build_everything.sh && \
    cd emsdk-portable

RUN  /bin/bash -c 'source buildscripts/emscripten/emsdk-portable/emsdk_env.sh && git clone https://github.com/EasyRPG/liblcf && cd liblcf && export EM_PKG_CONFIG_PATH=/workdir/buildscripts/emscripten/lib/pkgconfig && autoreconf -fi && emconfigure ./configure --prefix=/workdir/buildscripts/emscripten --disable-shared && make install'

RUN apt-get install -y ninja-build

# Build ynoclient
RUN /bin/bash -c 'source buildscripts/emscripten/emsdk-portable/emsdk_env.sh && git clone https://github.com/horahoradev/easyrpg_multi.git && ln -s /workdir /root/workdir && cd easyrpg_multi && ./cmake_build.sh && cd build && /usr/bin/ninja'


FROM ubuntu:rolling

WORKDIR /multi_server

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
	apt-get install -y git wget gcc && \
	mkdir public

RUN cd /usr/local && \
	wget https://golang.org/dl/go1.17.3.linux-amd64.tar.gz && \
	rm -rf /usr/local/go && \
	tar -C /usr/local -xzf go1.17.3.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

COPY orbs orbs

RUN cd orbs && \
	go mod vendor && \
    go build --mod=vendor -o /multi_server/multi_server .

RUN apt-get install -y python3 unzip python3-pip locales && \
	pip install gdown

RUN locale-gen ja_JP.UTF-8
ENV LANG ja_JP.UTF-8
ENV LANGUAGE ja_JP
ENV LC_ALL ja_JP.UTF-8

RUN gdown https://drive.google.com/uc?id=1-Oo8BsA1-bMFZoHuhfvvaASh1QJthrpk -O ./public/y2kki.zip && \
	cd public && \
	unzip ./y2kki.zip && \
	mkdir -p /multi_server/public/play/ && \
	/bin/bash -c 'mv /multi_server/public/*ver0.113* /multi_server/public/play/gamesdefault'

COPY gencache /multi_server/public/play/gamesdefault/ВфВ▀2В┴Вл/

RUN cd /multi_server/public/play/gamesdefault/ВфВ▀2В┴Вл/ && \
	./gencache

RUN /bin/bash -c 'mv /multi_server/public/play/gamesdefault/ВфВ▀2В┴Вл/* /multi_server/public/play/gamesdefault/'

COPY --from=0 /workdir/ynoclient/build/index.wasm /multi_server/public
COPY --from=0 /workdir/ynoclient/build/index.js /multi_server/public

COPY play.html /multi_server/public
COPY play.css /multi_server/public
COPY chat.js /multi_server/public

COPY index.html /multi_server/public


RUN apt-get install -y sudo

ENTRYPOINT ["sudo", "./multi_server"]

