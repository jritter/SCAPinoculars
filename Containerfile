# We do a two stage build
FROM golang:1.21 as builder
WORKDIR /build
COPY . .

# Do some GO optimization
ARG VERSION=main
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Let's build it! :-)
RUN go build -a -o scapinoculars .

# Now let's assemble the image
FROM registry.access.redhat.com/ubi9/ubi-minimal

# We need the openscap-scanner package to generate the fancy
# HTML reports
RUN microdnf install -y openscap-scanner

# Let's put everything in /opt/go because why not
WORKDIR /opt/go
# Copy the binary from the builder image
COPY --from=builder /build/scapinoculars .
# Copy the Go templates, but this time from the repository
COPY ./templates ./templates
# We are using port 2112, also because why not
EXPOSE 2112
# We don't need root privileges, yay!
USER 1001
# And we launch the binary! 
CMD ["/opt/go/scapinoculars"]
