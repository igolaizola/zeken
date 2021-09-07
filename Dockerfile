# builder image
FROM golang:1.16-alpine as builder
ARG TARGETPLATFORM
COPY . /src
WORKDIR /src
RUN apk add --no-cache make bash git
RUN make app-build PLATFORMS=$TARGETPLATFORM

# running image
FROM alpine:3.14
WORKDIR /home
COPY --from=builder /src/bin/zeken-* /bin/zeken

# executable
ENTRYPOINT [ "/bin/zeken" ]
