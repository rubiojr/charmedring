FROM scratch
COPY cring /usr/local/bin/cring

# Create /data directory
WORKDIR /data
# Expose data volume
VOLUME /data

EXPOSE 35351/tcp

# Set the default command
ENTRYPOINT [ "/usr/local/bin/cring" ]
CMD [ "serve" ]
