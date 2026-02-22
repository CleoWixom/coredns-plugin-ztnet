# ztnet

## Name

`ztnet` - resolve A/AAAA records for ZeroTier members from a ZTNET API endpoint.

## Description

The `ztnet` plugin periodically polls ZTNET network/member APIs and serves DNS answers from an in-memory cache.

For each authorized member in configured networks, the plugin serves:
- `A` records for member IPv4 assignments,
- `AAAA` records computed from RFC4193 and/or 6plane modes when enabled by network settings.

The plugin resolves both `<member-name>.<zone>` and `<member-id>.<zone>` names.

## Syntax

```corefile
ztnet {
    endpoint  http://localhost:3000
    token     <api-token>
    network   home.lan:8056c2e21c000001
    network   ztnet.network:abcdef01234567aa
    refresh   60s
    dns_ttl   30s
    fallthrough
}
```

- `endpoint` is required.
- `token` is optional when `ZTNET_API_TOKEN` is set.
- `network` is required and repeatable; format is `<zone>:<networkID>`.
- `refresh` and `dns_ttl` are optional durations.
- `fallthrough` is optional.

## Examples

```corefile
ztnet.network home.lan {
    ztnet {
        endpoint  http://localhost:3000
        network   home.lan:8056c2e21c000001
        refresh   60s
        dns_ttl   30s
        fallthrough
    }
}
```
