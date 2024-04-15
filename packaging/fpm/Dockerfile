FROM debian:12

RUN apt-get update
RUN apt-get install -y ruby ruby-dev rubygems build-essential git rpm

RUN gem install --no-document fpm -v 1.15.1

VOLUME /repo
WORKDIR /repo

ENV PACKAGE="deb"
ENV VERSION=""
ENV ARCH="amd64"
ENV OUTPUT_DIR="/repo/dist/"

CMD ./packaging/fpm/$PACKAGE/build.sh "$VERSION" "$ARCH" "$OUTPUT_DIR"
