FROM golang:1.21-alpine as builder
WORKDIR /src

COPY go.sum go.sum
COPY go.mod go.mod
RUN go mod download

COPY main.go main.go
COPY pkg pkg

RUN go build -o dbt-docs

FROM alpine:3
WORKDIR /app

COPY --from=builder /src/dbt-docs /app/dbt-docs
COPY templates /app/templates

CMD ["/app/dbt-docs"]
