FROM golang:1.26-alpine AS build

WORKDIR /src
# The release tag, logged at start so what is deployed is readable from the logs.
ARG VERSION=local
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY web/ ./web/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.buildVersion=${VERSION}" -o /out/muninn .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/muninn /usr/local/bin/muninn
EXPOSE 8095
CMD ["muninn"]
