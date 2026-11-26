# Changelog

## 1.0.0 (2026-11-26)

Initial release for Grafana 13 and Dependency-Track 5.x.

### Data source

- Portfolio and project metrics, as current values or time series, labelled with project, version and tags.
- Findings and policy violations as tables, or as per-project counts for alerting.
- Project inventory, SBOM age and vulnerability database metrics.
- Alerting support. Projects that never received an SBOM report an infinite age, so stale-SBOM rules fire for them.
- Annotations for BOM imports, new findings and policy violations.
- Template variables for projects, versions, tags, classifiers and policies.
- Links from tables to the Dependency-Track UI.
- Response cache, request concurrency limit, Prometheus metrics and tracing.

### App

- Overview, Projects (with per-project drill-down), Findings, Policy violations and SBOM freshness pages.
- Bundled dashboards: Portfolio overview, Project, Findings, Policy violations and SBOM freshness.

### Examples

- Alert rules in Grafana provisioning format.
- Terraform configuration for the data source and alert rules.

### Signature

- Signed with a temporary private signature for `http://localhost:3000/` while the plugin awaits the community signature level. Later releases are signed as community.
