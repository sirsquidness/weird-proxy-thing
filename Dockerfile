FROM golang:1.18

WORKDIR /usr/src/app

EXPOSE 1234

COPY . .
RUN go build -v -o /usr/local/bin/weird-proxy-thing -buildvcs=false

CMD ["weird-proxy-thing"]
