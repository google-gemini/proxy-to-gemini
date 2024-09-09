FROM golang AS builder
WORKDIR /go/code
ADD . /go/code
RUN CGO_ENABLED=0 go build -o /proxy ./cmd/proxy-to-gemini

FROM alpine:latest
COPY --from=builder /proxy /proxy-to-gemini
RUN apk --no-cache add ca-certificates \
  && update-ca-certificates
ENTRYPOINT ["/proxy-to-gemini"]
