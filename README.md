# Dependency-Track for Grafana

Visualize and alert on your software supply chain risk from [OWASP Dependency-Track](https://dependencytrack.org/).

The plugin is a Grafana app (`windkube-dependencytrack-app`) that bundles a backend data source (`windkube-dependencytrack-datasource`).

> [!WARNING]
> This plugin is under active development and not yet published to the Grafana plugin catalog.

## Requirements

- Grafana 13.0 or later
- Dependency-Track 5.x
- A Dependency-Track API key (team API key or service account) with the `VIEW_PORTFOLIO`, `VIEW_VULNERABILITY` and `VIEW_POLICY_VIOLATION` permissions

## Contributing

See [CONTRIBUTING.md](https://github.com/WindKube/grafana-plugin-dependency-track/blob/main/CONTRIBUTING.md).

## License

[Apache License 2.0](https://github.com/WindKube/grafana-plugin-dependency-track/blob/main/LICENSE)
