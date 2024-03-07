FROM golang:bookworm as build-stage

WORKDIR /app

COPY . ./

RUN go mod download

# Build CSS
RUN curl -sL --output tailwindcss  https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.1/tailwindcss-linux-x64
RUN chmod +x tailwindcss
RUN ./tailwindcss -i main.css -o ./assets/css/main.css --minify

# Build Templates
RUN go install github.com/a-h/templ/cmd/templ@v0
RUN templ generate

# We need to build the binaries for duckdb CGO_ENABLED=1
RUN CGO_ENABLED=1 GOOS=linux go build -o /gbif-extinct

FROM debian:bookworm as production-stage

RUN apt-get update && apt-get install ca-certificates -y && update-ca-certificates

WORKDIR /

COPY --from=build-stage /gbif-extinct /gbif-extinct
COPY --from=build-stage /app/assets /assets
COPY --from=build-stage /app/migrations /migrations
COPY --from=build-stage /app/README.md /README.md

EXPOSE 1323

CMD ["./gbif-extinct"]