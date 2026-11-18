FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /server ./cmd/server

FROM alpine:3.22
WORKDIR /app
COPY --from=build /server /app/server
COPY web /app/web
USER 10001:10001
ENV ADDR=0.0.0.0:8080
EXPOSE 8080
CMD ["/app/server"]
