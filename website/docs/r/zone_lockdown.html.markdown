---
layout: "cloudflare"
page_title: "Cloudflare: cloudflare_zone_lockdown"
sidebar_current: "docs-cloudflare-resource-zone-lockdown"
description: |-
  Provides a Cloudflare resource to lock down access to URLs by IP address or IP ranges.
---

# cloudflare_zone_lockdown

Provides a Cloudflare Zone Lockdown resource. Zone Lockdown allows you to define one or more URLs (with wildcard matching on the domain or path) that will only permit access if the request originates from an IP address that matches a safelist of one or more IP addresses and/or IP ranges.

## Example Usage

```hcl
# Restrict access to these endpoints to requests from a known IP address.
resource "cloudflare_zone_lockdown" "endpoint_lockdown" {
  zone        = "api.mysite.com"
  paused      = "false"
  description = "Restrict access to these endpoints to requests from a known IP address"
  urls = [
    "api.mysite.com/some/endpoint*",
  ]
  configurations = [
    {
      "target" = "ip"
      "value" = "198.51.100.4"
    },
  ]
}
```

## Argument Reference

The following arguments are supported:

* `zone` - The DNS zone to which the lockdown will be added. Will be resolved to `zone_id` upon creation.
* `description` - (Optional) A descriptionabout the lockdown entry. Typically used as a reminder or explanation for the lockdown.
* `urls` - (Required) A list of simple wildcard patterns to match requests against.
* `configurations` - (Required) A list of IP addresses or IP ranges to match the request against.  IP addresses should just be standard IPv4 notation i.e. "198.51.100.4" and IP ranges limited to /16 and /24 i.e. "198.51.100.4/16".

## Attributes Reference

The following attributes are exported:

* `id` - The access rule ID.
* `zone_id` - The DNS zone ID.

## Import

Records can be imported using a composite ID formed of zone name and record ID, e.g.

```
$ terraform import cloudflare_zone_lockdown  api.mysite.com/d41d8cd98f00b204e9800998ecf8427e
```

where:

* `d41d8cd98f00b204e9800998ecf8427e` - zone lockdown ID as returned by [API](https://api.cloudflare.com/#zone-lockdown-list-lockdown-rules)
