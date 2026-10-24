# Godwit API server
#
# Build: docker build -t godwit .
# Run:   docker run -p 127.0.0.1:8080:8080 -e GODWIT_API_KEYS=change-me godwit

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/godwit-engine ./server

# Empty base image: only the server binary and the model, no shell or OS packages
FROM scratch
COPY --from=build /out/godwit-engine /app/godwit-engine
COPY models/all-MiniLM-L6-v2 /app/models/all-MiniLM-L6-v2

ENV GODWIT_ADDR=0.0.0.0:8080
EXPOSE 8080
USER 65532:65532

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/app/godwit-engine", "-healthcheck"]

ENTRYPOINT ["/app/godwit-engine"]
