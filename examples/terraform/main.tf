terraform {
  required_version = ">= 1.6"
  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "~> 4.0"
    }
  }
}

provider "grafana" {
  url  = var.grafana_url
  auth = var.grafana_auth
}

resource "grafana_data_source" "dependency_track" {
  type = "windkube-dependencytrack-datasource"
  name = "Dependency-Track"
  url  = var.dependency_track_url

  json_data_encoded = jsonencode({
    uiUrl                 = var.dependency_track_ui_url
    cacheTtl              = 60
    maxConcurrentRequests = 10
  })

  secure_json_data_encoded = jsonencode({
    apiKey = var.dependency_track_api_key
  })
}

resource "grafana_folder" "dependency_track" {
  title = "Dependency-Track"
}

resource "grafana_rule_group" "dependency_track" {
  name             = "dependency-track"
  folder_uid       = grafana_folder.dependency_track.uid
  interval_seconds = 300

  rule {
    name           = "Critical vulnerabilities in project"
    condition      = "C"
    for            = "0s"
    no_data_state  = "OK"
    exec_err_state = "Error"
    labels         = { source = "dependency-track" }
    annotations = {
      summary = "{{ $labels.project }} {{ $labels.version }} has {{ $values.B }} critical vulnerabilities"
    }

    data {
      ref_id         = "A"
      datasource_uid = grafana_data_source.dependency_track.uid
      relative_time_range {
        from = 600
        to   = 0
      }
      model = jsonencode({
        refId     = "A"
        queryType = "projectMetrics"
        mode      = "current"
        metrics   = ["critical"]
        projects  = { mode = "all", onlyLatest = true }
      })
    }
    data {
      ref_id         = "B"
      datasource_uid = "__expr__"
      relative_time_range {
        from = 0
        to   = 0
      }
      model = jsonencode({ refId = "B", type = "reduce", expression = "A", reducer = "last" })
    }
    data {
      ref_id         = "C"
      datasource_uid = "__expr__"
      relative_time_range {
        from = 0
        to   = 0
      }
      model = jsonencode({
        refId      = "C"
        type       = "threshold"
        expression = "B"
        conditions = [{ evaluator = { type = "gt", params = [0] } }]
      })
    }
  }

  rule {
    name           = "SBOM not updated in ${var.stale_sbom_days} days"
    condition      = "C"
    for            = "0s"
    no_data_state  = "OK"
    exec_err_state = "Error"
    labels         = { source = "dependency-track" }
    annotations = {
      summary = "{{ $labels.project }} {{ $labels.version }} has no SBOM upload in the last ${var.stale_sbom_days} days"
    }

    data {
      ref_id         = "A"
      datasource_uid = grafana_data_source.dependency_track.uid
      relative_time_range {
        from = 600
        to   = 0
      }
      model = jsonencode({
        refId     = "A"
        queryType = "projects"
        format    = "bomAge"
        projects  = { mode = "all", onlyLatest = true }
      })
    }
    data {
      ref_id         = "B"
      datasource_uid = "__expr__"
      relative_time_range {
        from = 0
        to   = 0
      }
      model = jsonencode({ refId = "B", type = "reduce", expression = "A", reducer = "last" })
    }
    data {
      ref_id         = "C"
      datasource_uid = "__expr__"
      relative_time_range {
        from = 0
        to   = 0
      }
      model = jsonencode({
        refId      = "C"
        type       = "threshold"
        expression = "B"
        conditions = [{ evaluator = { type = "gt", params = [var.stale_sbom_days * 86400] } }]
      })
    }
  }
}
