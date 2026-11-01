# Builds goilerplate.com: the static pages and the small server for them.
# Build from the repository root: docker build -f site/Dockerfile .
FROM golang:1.26 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# templ pinned by the tool directive in go.mod, Tailwind by version.
RUN go tool templ generate -path site
RUN ARCH=$([ "$(uname -m)" = "aarch64" ] && echo arm64 || echo x64) && \
  curl --fail --location --silent --show-error --output /usr/local/bin/tailwindcss \
    https://github.com/tailwindlabs/tailwindcss/releases/download/v4.3.3/tailwindcss-linux-$ARCH && \
  chmod +x /usr/local/bin/tailwindcss && \
  tailwindcss -i site/assets/css/globals.css -o site/assets/css/output.css --minify

RUN go run ./site/cmd/build -out /out/dist
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./site/cmd/server

FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/dist /app/dist

EXPOSE 8090
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=6 CMD ["/app/server", "healthcheck"]
CMD ["/app/server", "-dir", "/app/dist"]
