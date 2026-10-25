FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ironbot ./cmd/ironbot

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/ironbot /ironbot
USER nonroot:nonroot
ENTRYPOINT ["/ironbot"]
