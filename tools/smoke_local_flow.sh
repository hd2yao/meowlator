#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
CAT_ID="${CAT_ID:-cat-default}"
SCENE_TAG="${SCENE_TAG:-UNKNOWN}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

LOGIN_JSON="$TMP_DIR/login.json"
UPLOAD_URL_JSON="$TMP_DIR/upload_url.json"
UPLOAD_RESP_JSON="$TMP_DIR/upload_resp.json"
FINALIZE_JSON="$TMP_DIR/finalize.json"
FEEDBACK_JSON="$TMP_DIR/feedback.json"
DELETE_JSON="$TMP_DIR/delete.json"
SAMPLE_FILE="$TMP_DIR/smoke.jpg"

printf 'smoke-image-payload\n' > "$SAMPLE_FILE"

log() {
  printf '[smoke] %s\n' "$*"
}

require_json_field() {
  local file="$1"
  local expr="$2"
  local value
  value="$(jq -er "$expr" "$file")" || {
    printf 'failed to parse %s from %s\n' "$expr" "$file" >&2
    cat "$file" >&2
    exit 1
  }
  printf '%s' "$value"
}

compute_sig() {
  local method="$1"
  local path="$2"
  local ts="$3"
  local body="$4"
  local token="$5"
  python3 - "$method" "$path" "$ts" "$body" "$token" <<'PY'
import sys

method, path, ts, body, token = sys.argv[1:]
value = f"{method}|{path}|{ts}|{body}|{token}"
hash_value = 0x811C9DC5
for char in value:
    hash_value ^= ord(char)
    hash_value = (hash_value * 0x01000193) & 0xFFFFFFFF
print(f"{hash_value:08x}")
PY
}

signed_post_json() {
  local path="$1"
  local body="$2"
  local output="$3"
  local ts sig
  ts="$(date +%s)"
  sig="$(compute_sig "POST" "$path" "$ts" "$body" "$SESSION_TOKEN")"
  curl -sSf \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $SESSION_TOKEN" \
    -H "X-User-Id: $USER_ID" \
    -H "X-Req-Ts: $ts" \
    -H "X-Req-Sig: $sig" \
    -X POST \
    -d "$body" \
    "$BASE_URL$path" > "$output"
}

signed_delete() {
  local path="$1"
  local output="$2"
  local ts sig
  ts="$(date +%s)"
  sig="$(compute_sig "DELETE" "$path" "$ts" "" "$SESSION_TOKEN")"
  curl -sSf \
    -H "Authorization: Bearer $SESSION_TOKEN" \
    -H "X-User-Id: $USER_ID" \
    -H "X-Req-Ts: $ts" \
    -H "X-Req-Sig: $sig" \
    -X DELETE \
    "$BASE_URL$path" > "$output"
}

log "login"
LOGIN_BODY="$(jq -nc --arg code "smoke-$(date +%s)" '{code: $code}')"
curl -sSf \
  -H "Content-Type: application/json" \
  -X POST \
  -d "$LOGIN_BODY" \
  "$BASE_URL/v1/auth/wechat/login" > "$LOGIN_JSON"

USER_ID="$(require_json_field "$LOGIN_JSON" '.userId')"
SESSION_TOKEN="$(require_json_field "$LOGIN_JSON" '.sessionToken')"
log "session userId=$USER_ID"

log "request upload-url"
UPLOAD_URL_BODY="$(jq -nc --arg catId "$CAT_ID" '{catId: $catId, suffix: ".jpg"}')"
signed_post_json "/v1/samples/upload-url" "$UPLOAD_URL_BODY" "$UPLOAD_URL_JSON"

SAMPLE_ID="$(require_json_field "$UPLOAD_URL_JSON" '.sampleId')"
UPLOAD_URL="$(require_json_field "$UPLOAD_URL_JSON" '.uploadUrl')"
log "sampleId=$SAMPLE_ID"

log "upload file"
curl -sSf \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "X-User-Id: $USER_ID" \
  -F "file=@$SAMPLE_FILE;type=image/jpeg" \
  "$UPLOAD_URL" > "$UPLOAD_RESP_JSON"

STORED_AT="$(require_json_field "$UPLOAD_RESP_JSON" '.storedAt')"
log "storedAt=$STORED_AT"

log "finalize inference"
FINALIZE_BODY="$(jq -nc --arg sampleId "$SAMPLE_ID" --arg sceneTag "$SCENE_TAG" '{sampleId: $sampleId, deviceCapable: false, sceneTag: $sceneTag}')"
curl -sSf \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "X-User-Id: $USER_ID" \
  -X POST \
  -d "$FINALIZE_BODY" \
  "$BASE_URL/v1/inference/finalize" > "$FINALIZE_JSON"

TOP_LABEL="$(require_json_field "$FINALIZE_JSON" '.result.intentTop3[0].label')"
FALLBACK_USED="$(require_json_field "$FINALIZE_JSON" '.fallbackUsed')"
NEED_FEEDBACK="$(require_json_field "$FINALIZE_JSON" '.needFeedback')"
log "topIntent=$TOP_LABEL fallbackUsed=$FALLBACK_USED needFeedback=$NEED_FEEDBACK"

log "submit feedback"
FEEDBACK_BODY="$(jq -nc --arg sampleId "$SAMPLE_ID" '{sampleId: $sampleId, isCorrect: true}')"
curl -sSf \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "X-User-Id: $USER_ID" \
  -X POST \
  -d "$FEEDBACK_BODY" \
  "$BASE_URL/v1/feedback" > "$FEEDBACK_JSON"

FEEDBACK_ID="$(require_json_field "$FEEDBACK_JSON" '.feedbackId')"
TRAINING_WEIGHT="$(require_json_field "$FEEDBACK_JSON" '.trainingWeight')"
log "feedbackId=$FEEDBACK_ID trainingWeight=$TRAINING_WEIGHT"

log "cleanup sample"
signed_delete "/v1/samples/$SAMPLE_ID" "$DELETE_JSON"
DELETED_ID="$(require_json_field "$DELETE_JSON" '.deleted')"
log "deleted sampleId=$DELETED_ID"

log "done"
