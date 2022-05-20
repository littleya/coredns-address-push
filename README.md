# Address Push

## Name

Plugin *AddressPush*

## Description

The plugin need configure after forward plugin, e.g.

``` conf
address_push:github.com/littleya/coredns-address-push
forward:forward
```

## Syntax

``` conf
{
    address_push {
        enabled     [true|false]
        type        [ipset|netmg|routeros|vyos]
        host        [host:port]
        auth_user   [username for auth]
        auth_key    [password or api key for auth]
        ipv4        [ipv4_address_list]
        ipv6        [ipv6_address_list]
    }
}
```

## Examples
