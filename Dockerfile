# builder image
FROM golang:alpine as builder
COPY . /src
WORKDIR /src
RUN apk add --no-cache make bash
RUN make build

# running image
FROM alpine
WORKDIR /home
COPY --from=builder /src/bin/zeken /bin/zeken

# executable
ENTRYPOINT [ "/bin/zeken" ]
