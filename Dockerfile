FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
ENV PATH=/bin
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY cring /usr/local/bin/cring

# Create /data directory
WORKDIR /data
# Expose data volume
VOLUME /data

EXPOSE 35351/tcp


# Set the default command
ENTRYPOINT [ "/usr/local/bin/cring" ]
CMD [ "serve" ]

