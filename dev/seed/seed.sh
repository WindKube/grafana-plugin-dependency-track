#!/bin/sh
# Seeds a fresh Dependency-Track 5.x instance for local development and e2e tests.
# Safe to re-run after a partial failure: existing teams, policies and projects are reused.
set -eu
set -o pipefail

DT_URL="${DT_URL:-http://apiserver:8080}"
ADMIN_PASSWORD="${DT_ADMIN_PASSWORD:-grafana-dev}"
OUT_DIR="${OUT_DIR:-/seed-out}"
SEED_DIR="$(cd "$(dirname "$0")" && pwd)"
API="$DT_URL/api/v1"

log() { echo "[seed] $*"; }
fail() {
  echo "[seed] ERROR: $*" >&2
  exit 1
}

# require <description> <value>: fails when a lookup returned nothing.
require() {
  if [ -z "$2" ] || [ "$2" = "null" ]; then
    fail "$1 is empty"
  fi
  printf '%s' "$2"
}

# wait_until <description> <attempts> <command...>: polls once per second.
wait_until() {
  desc="$1"
  attempts="$2"
  shift 2
  i=0
  until "$@"; do
    i=$((i + 1))
    [ "$i" -lt "$attempts" ] || fail "timed out waiting for $desc"
    sleep 1
  done
}

if [ -s "$OUT_DIR/api-key" ]; then
  log "already seeded, API key present in $OUT_DIR/api-key"
  exit 0
fi

log "waiting for $DT_URL"
dt_ready() { curl -sf "$DT_URL/api/version" >/dev/null; }
wait_until "Dependency-Track to start" 300 dt_ready

curl -sf -o /dev/null -X POST "$API/user/forceChangePassword" \
  --data-urlencode username=admin --data-urlencode password=admin \
  --data-urlencode "newPassword=$ADMIN_PASSWORD" --data-urlencode "confirmPassword=$ADMIN_PASSWORD" || true
JWT="$(curl -sf -X POST "$API/user/login" --data-urlencode username=admin --data-urlencode "password=$ADMIN_PASSWORD")" ||
  fail "admin login failed"

dt() {
  method="$1"
  path="$2"
  shift 2
  curl -sSf -X "$method" "$API$path" -H "Authorization: Bearer $JWT" -H 'Content-Type: application/json' "$@"
}

log "creating team and API key"
TEAM_UUID="$(dt GET '/team?pageSize=100' | jq -r '.[] | select(.name=="Grafana") | .uuid' | head -n1)"
if [ -z "$TEAM_UUID" ]; then
  TEAM_UUID="$(dt PUT /team -d '{"name":"Grafana"}' | jq -r .uuid)"
fi
TEAM_UUID="$(require "team uuid" "$TEAM_UUID")"
for p in VIEW_PORTFOLIO VIEW_VULNERABILITY VIEW_POLICY_VIOLATION; do
  dt POST "/permission/$p/team/$TEAM_UUID" >/dev/null
done
API_KEY="$(require "API key" "$(dt PUT "/team/$TEAM_UUID/key" | jq -r .key)")"

log "creating policies"
create_policy() {
  name="$1" violation_state="$2" subject="$3" operator="$4" value="$5" violation_type="$6"
  existing="$(dt GET '/policy?pageSize=100' | jq -c --arg n "$name" '.[] | select(.name==$n)')"
  if [ -n "$existing" ]; then
    uuid="$(echo "$existing" | jq -r .uuid)"
    [ "$(echo "$existing" | jq '.policyConditions // [] | length')" -eq 0 ] || return 0
  else
    uuid="$(dt PUT /policy -d "{\"name\":\"$name\",\"operator\":\"ANY\",\"violationState\":\"$violation_state\"}" | jq -r .uuid)"
  fi
  uuid="$(require "policy $name uuid" "$uuid")"
  dt PUT "/policy/$uuid/condition" \
    -d "{\"subject\":\"$subject\",\"operator\":\"$operator\",\"value\":\"$value\",\"violationType\":\"$violation_type\"}" >/dev/null
}
create_policy "Banned packages" FAIL PACKAGE_URL MATCHES 'pkg:npm/(moment|marked)@.*' OPERATIONAL
create_policy "Embedded databases" WARN PACKAGE_URL MATCHES 'pkg:maven/com.h2database/.*' OPERATIONAL
create_policy "Python review" INFO PACKAGE_URL MATCHES 'pkg:pypi/.*' OPERATIONAL

# event_done <token>: true once Dependency-Track has finished the work behind an event token.
event_done() { [ "$(dt GET "/event/token/$1" | jq -r .processing)" = "false" ]; }

log "uploading BOMs"
for bom in "$SEED_DIR"/boms/*.cdx.json; do
  base="$(basename "$bom" .cdx.json)"
  name="${base%-*}"
  version="${base##*-}"
  tags="$(jq -c '[.metadata.properties // [] | .[] | select(.name=="seed:tag") | {name: .value}]' "$bom")"
  latest=false
  [ "$base" = "payments-api-1.4.0" ] || latest=true
  token="$(jq -n --arg n "$name" --arg v "$version" --argjson t "$tags" --argjson l "$latest" --rawfile b "$bom" \
    '{projectName: $n, projectVersion: $v, autoCreate: true, isLatest: $l, projectTags: $t, bom: ($b | @base64)}' |
    dt PUT /bom -d @- | jq -r .token)"
  token="$(require "BOM upload token for $base" "$token")"
  wait_until "BOM $base to be processed" 120 event_done "$token"
  log "  $name $version"
done

log "creating internal vulnerabilities"
add_vuln() {
  id="$1" severity="$2" cvss="$3" purl="$4"
  vuln_uuid="$(dt GET "/vulnerability/source/INTERNAL/vuln/$id" 2>/dev/null | jq -r .uuid || true)"
  if [ -z "$vuln_uuid" ] || [ "$vuln_uuid" = "null" ]; then
    vuln_uuid="$(dt PUT /vulnerability -d "{\"vulnId\":\"$id\",\"source\":\"INTERNAL\",\"title\":\"$id in $purl\",\"description\":\"Seeded vulnerability for development.\",\"severity\":\"$severity\",\"cvssV3BaseScore\":$cvss}" | jq -r .uuid)"
  fi
  vuln_uuid="$(require "vulnerability $id uuid" "$vuln_uuid")"
  dt GET "/component/identity?purl=$(jq -rn --arg p "$purl" '$p|@uri')&pageSize=100" | jq -r '.[].uuid' |
    while read -r component; do
      dt POST "/vulnerability/$vuln_uuid/component/$component" >/dev/null
    done
}
add_vuln INT-2021-0001 CRITICAL 10.0 'pkg:maven/org.apache.logging.log4j/log4j-core@2.14.1'
add_vuln INT-2021-0002 CRITICAL 9.8 'pkg:npm/minimist@1.2.5'
add_vuln INT-2020-0003 HIGH 7.4 'pkg:npm/lodash@4.17.15'
add_vuln INT-2022-0004 HIGH 7.5 'pkg:npm/jsonwebtoken@8.5.1'
add_vuln INT-2021-0005 MEDIUM 6.5 'pkg:npm/axios@0.21.1'
add_vuln INT-2017-0006 MEDIUM 5.3 'pkg:npm/marked@0.3.6'
add_vuln INT-2021-0007 HIGH 7.5 'pkg:npm/ua-parser-js@0.7.28'
add_vuln INT-2020-0008 CRITICAL 9.8 'pkg:maven/com.fasterxml.jackson.core/jackson-databind@2.9.10.1'
add_vuln INT-2022-0009 MEDIUM 6.5 'pkg:maven/org.yaml/snakeyaml@1.29'
add_vuln INT-2021-0010 HIGH 8.8 'pkg:pypi/pyyaml@5.3.1'
add_vuln INT-2023-0011 MEDIUM 5.4 'pkg:pypi/requests@2.25.1'
add_vuln INT-2023-0012 LOW 3.7 'pkg:pypi/urllib3@1.26.4'
add_vuln INT-2022-0013 LOW 2.0 'pkg:npm/moment@2.29.1'

log "evaluating policies and refreshing metrics"
PROJECTS="$(dt GET '/project?pageSize=100' | jq -r '.[].uuid')"
for project in $PROJECTS; do
  token="$(require "analysis token for $project" "$(dt POST "/finding/project/$project/analyze" | jq -r .token)")"
  wait_until "analysis of $project" 120 event_done "$token"
done
for project in $PROJECTS; do
  dt GET "/metrics/project/$project/refresh" >/dev/null
done
dt GET /metrics/portfolio/refresh >/dev/null

mkdir -p "$OUT_DIR"
printf '%s' "$API_KEY" >"$OUT_DIR/api-key"
log "done"
