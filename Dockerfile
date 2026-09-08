# We do a two stage build
FROM registry.access.redhat.com/hi/go:1.27.0-builder as builder
WORKDIR /build
COPY . .

# Do some GO optimization
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux

# Let's build it! :-)
RUN go build -a -o scapinoculars .

# Now let's assemble the image
FROM registry.access.redhat.com/hi/openscap:1.4.4
ARG HASH=unknown
ARG VERSION=unknown
ENV BUILD_HASH=${HASH}
ENV BUILD_VERSION=${VERSION}

# Let's put everything in /opt/go because why not
WORKDIR /opt/go
# Copy the binary from the builder image
COPY --from=builder /build/scapinoculars .
# Copy the Go templates, but this time from the repository
COPY ./templates ./templates
# Copy CSS, so that the reports are beautiful in offline environments
ADD --chmod=644 https://cdn.jsdelivr.net/npm/bootstrap@3.4.1/dist/css/bootstrap.min.css ./styles/
# We are using port 2112, also because why not
EXPOSE 2112
# We don't need root privileges, yay!
USER 1001
# And we launch the binary! 
ENTRYPOINT ["/opt/go/scapinoculars"]
