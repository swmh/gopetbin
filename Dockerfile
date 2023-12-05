FROM golang:1.21.3-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o gopetbin cmd/gopetbin/main.go
RUN CGO_ENABLED=0 go build -o clean cmd/clean/main.go

FROM scratch

WORKDIR /app

COPY --from=builder /tmp /tmp
COPY --from=builder /build/gopetbin ./
COPY --from=builder /build/clean ./
COPY ./config/config-sample.yml ./

EXPOSE 80

CMD ["./gopetbin"]
