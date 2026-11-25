variable "grafana_url" {
  description = "Grafana base URL."
  type        = string
}

variable "grafana_auth" {
  description = "Grafana service account token."
  type        = string
  sensitive   = true
}

variable "dependency_track_url" {
  description = "Dependency-Track API server URL, for example https://dtrack-api.example.com."
  type        = string
}

variable "dependency_track_ui_url" {
  description = "Dependency-Track frontend URL, used for links from tables. Leave empty to disable links."
  type        = string
  default     = ""
}

variable "dependency_track_api_key" {
  description = "API key of a Dependency-Track team with VIEW_PORTFOLIO, VIEW_VULNERABILITY and VIEW_POLICY_VIOLATION."
  type        = string
  sensitive   = true
}

variable "stale_sbom_days" {
  description = "Alert when a project has had no SBOM upload for this many days."
  type        = number
  default     = 30
}
