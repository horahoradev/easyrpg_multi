FROM ubuntu:rolling

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
	apt-get install -y git wget gcc && \
	mkdir -p /multi_server/public/play

WORKDIR /multi_server

RUN apt-get install -y python3 unzip python3-pip locales locales-all && \
	pip install gdown

RUN locale-gen ja_JP.UTF-8
ENV LANG ja_JP.UTF-8
ENV LANGUAGE ja_JP
ENV LC_ALL ja_JP.UTF-8

RUN gdown https://drive.google.com/uc?id=1c8g2XBLFQ6L6KNrmI3njhgk714uX0p3W -O ./public/y2kki.zip && \
	cd public && \
	unzip -O shift-jis ./y2kki.zip && \
	mkdir -p /multi_server/public/play/ && \
	/bin/bash -c 'mv /multi_server/public/ゆめ2っきver0.117g /multi_server/public/play/gamesdefault'

COPY gencache /multi_server/public/play/gamesdefault/ゆめ2っき/

RUN mv /multi_server/public/play/gamesdefault/ゆめ2っき/music /multi_server/public/play/gamesdefault/ゆめ2っき/Music

RUN cd /multi_server/public/play/gamesdefault/ゆめ2っき/ && \
	./gencache

RUN /bin/bash -c 'mv /multi_server/public/play/gamesdefault/ゆめ2っき/* /multi_server/public/play/gamesdefault/'

FROM ghcr.io/horahoradev/liblcf:master_liblcf

COPY ynoclient ynoclient

# Build ynoclient
RUN /bin/bash -c 'source buildscripts/emscripten/emsdk-portable/emsdk_env.sh && \
	ln -s /workdir /root/workdir && \
	cd ynoclient && \
	./cmake_build.sh && cd build && \
	/usr/bin/ninja && \
	echo "done"'

COPY ynoclient_modified ynoclient

## Recompile just the diffs lol
RUN  /bin/bash -c 'source buildscripts/emscripten/emsdk-portable/emsdk_env.sh && \
	ln -s /workdir /root/workdir && \
	cd ynoclient && \
	./cmake_build.sh && cd build && \
	/usr/bin/ninja && \
	echo "done"'

FROM nginx:mainline

RUN apt-get update && apt-get install -y locales locales-all

RUN locale-gen ja_JP.UTF-8
ENV LANG ja_JP.UTF-8
ENV LANGUAGE ja_JP
ENV LC_ALL ja_JP.UTF-8

COPY ynofront /usr/share/nginx/html

COPY --from=0 /multi_server/public/play/gamesdefault/ /usr/share/nginx/html/data/2kki

COPY --from=1 /workdir/ynoclient/build/index.wasm /usr/share/nginx/html/2kki
COPY --from=1 /workdir/ynoclient/build/index.js /usr/share/nginx/html/2kki
