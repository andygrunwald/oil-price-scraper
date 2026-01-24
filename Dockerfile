# Runtime image - using alpine for CA certificates
FROM alpine:3.21

# Install CA certificates for HTTPS requests
RUN apk add --no-cache ca-certificates

# Build args (passed by goreleaser)
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
ARG TARGETPLATFORM

# Copy the pre-built binary from goreleaser's build context
COPY ${TARGETPLATFORM}/oilscraper /oilscraper

# Expose metrics/status port
EXPOSE 8080

ENTRYPOINT ["/oilscraper"]
CMD ["run"]
