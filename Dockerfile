FROM golang:1.21.6 AS builder

WORKDIR /cpuset-plugin

COPY go.* ./

RUN go mod download

COPY cmd cmd

COPY pkg pkg

RUN CGO_ENABLED=0 go build -o bin/cpuset-plugin ./cmd/main.go

FROM ubuntu

WORKDIR /

COPY --from=builder /cpuset-plugin/bin/cpuset-plugin .

ENTRYPOINT ["./cpuset-plugin"]
