FROM golang AS builder
WORKDIR /go/code
ADD . /go/code
RUN CGO_ENABLED=0 go build -o /proxy ./cmd/proxy-to-gemini

FROM scratch
COPY --from=builder /proxy /proxy-to-gemini
ENTRYPOINT ["/proxy-to-gemini"]
