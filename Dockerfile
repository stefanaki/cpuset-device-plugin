FROM ubuntu

COPY main /usr/local/bin/main

ENTRYPOINT ["/usr/local/bin/main"]