# About

ping-dashboard is a simple dashboard to quickly check if a large amount of hosts are up (via ICMP).

# Building

```bash
$ cd /path/to/build/directory
$ GOBIN="$(pwd)" go install "github.com/korylprince/ping-dashboard@<tagged version>"
$ ./ping-dashboard
```

# Configuring

ping-dashboard is configured with environment variables:

Variable | Description | Default
-------- | ----------- | -------
HOSTSPATH | Path to hosts configuration (See Schema) | Must be configured
PINGERS | Number of concurrent pingers to use | runtime.NumCPU() * 2
RESOLVERS | Number of concurrent resolvers to use | runtime.NumCPU() * 4
QUEUESIZE | Size of pending ping/resolve queue | 1024
TIMEOUT | Duration to wait for an ICMP echo response | 1 second
USERNAME | Username for Basic Auth | admin
PASSWORD | Password for Basic Auth. If using the prebuilt Docker container, you can also specify PASSWORD_FILE for use with Docker secrets | Must be configured
PROXYHEADERS | Set to `true` if you want the server to rewrite IP addresses with X-Forwarded-For, etc headers | false
LISTENADDR | The host:port address you want the server to listen on | :80

# Schema

HOSTSPATH should point to a yaml file with the following schema:

```yaml
- category: Category 1
  hosts:
    - host1.example.com
    - host2.example.com
- category: Category 2
  hosts:
    - host3.example.com
    - host4.example.com
```

# Deploying

ping-dashboard is intended to be deployed behind a reverse proxy with TLS termination (e.g. traefik, nginx, etc). Don't forget to set PROXYHEADERS to true if doing so.

There's a prebuilt Docker container at `korylprince/ping-dashboard:<tagged version>`.
