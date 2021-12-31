FROM golang:1.17-alpine3.15 as builder
WORKDIR /app/
COPY . .
ARG opts
ARG arch
RUN go mod tidy
RUN env $opts $arch go build -o simple main.go

FROM alpine:3.15.0
ARG arch
LABEL arch=$arch
LABEL maintainer="Engin Diri <info@ediri.de>"
LABEL description="Simple HTTP server with a lofi gif as response"
COPY --from=builder /app/simple /app/simple
RUN adduser -D simple
RUN chown -R simple:simple /app
USER simple
CMD ["/app/simple"]