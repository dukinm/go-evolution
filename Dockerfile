FROM golang:alpine AS builder

WORKDIR /build

ADD go.mod .

COPY . .

RUN go build -o workfile main.go

FROM alpine

WORKDIR /build

COPY --from=builder /build/workfile /build/workfile

CMD ["./workfile"]