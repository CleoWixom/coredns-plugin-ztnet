# ztnet plugin

The `ztnet` plugin resolves A and AAAA records for ZeroTier members from a ZTNET API endpoint.

Example configuration:

```corefile
ztnet {
    endpoint  http://localhost:3000
    token     my-token
    network   home.lan:8056c2e21c000001
    refresh   60s
    dns_ttl   30s
    fallthrough
}
```
