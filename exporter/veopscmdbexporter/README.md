# Veops CMDB Exporter
<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [alpha]: logs   |
| Distributions | [contrib] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Aexporter%2Fveopscmdb%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Aexporter%2Fveopscmdb) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Aexporter%2Fveopscmdb%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Aexporter%2Fveopscmdb) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    | [@JaredTan95](https://www.github.com/JaredTan95) |

[alpha]: https://github.com/open-telemetry/opentelemetry-collector#alpha
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
<!-- end autogenerated section -->

This exporter allows creating [markers](https://docs.honeycomb.io/working-with-your-data/markers/), via the [Honeycomb Markers API](https://docs.honeycomb.io/api/tag/Markers#operation/createMarker), based on the look of incoming telemetry. 

The following configuration options are supported:

* `api_key` (Required): This is the API key for your CMDB account.
* `api_secret` (Required): This is the API secret for your CMDB account.
* `api_address` (Optional): This sets the hostname to send data to. If not set, will default to `http://localhost:5000`
* `ci_matches` (Required): This is a list of configurations to create an CI. 
  * `resouce_name`: (Required): Specifies the kubernetes resoruce type name(following `k8sobjectsreceiver`).
  * `ci_type` (Required): Specifies the marker type.
Example:
```yaml
exporters:
  veopscmdbexporter:
    api_address: "http://localhost:5000"
    api_key: "test-apikey"
    api_secret: "test-apisecret"
    kubernetes_cluster_ci_type: 58
```