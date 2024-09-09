FROM golang AS builder
WORKDIR /go/src/proxy-to-gemini
ADD . ./
RUN CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o /proxy-to-gemini

FROM scratch
COPY --from=builder /proxy-to-gemini /proxy-to-gemini
ENTRYPOINT ["/proxy-to-gemini"]
