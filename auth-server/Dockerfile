# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
ARG TARGETARCH
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/workit-auth-server .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/workit-auth-server /app/workit-auth-server

ENV WK_PUBLIC_BASE_URL=""
ENV WK_M365_CLIENT_ID=""
ENV WK_M365_TENANT_ID="organizations"
ENV WK_M365_BROKER_TOKEN=""
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/workit-auth-server"]
