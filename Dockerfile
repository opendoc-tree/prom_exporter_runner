# Build
FROM golang AS Build
WORKDIR /src
COPY . .
RUN go build

# Deploy
FROM golang
COPY --from=Build /src/prom_exporter_runner /prom_exporter_runner
USER nobody
ENTRYPOINT ["/prom_exporter_runner"]

# Build docker image
# docker build --force-rm -t dipakdock/prom_exporter_runner .
