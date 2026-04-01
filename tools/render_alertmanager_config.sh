#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE_PATH="${ROOT_DIR}/infra/monitoring/alertmanager.yml"
RUNTIME_PATH="${ROOT_DIR}/infra/monitoring/alertmanager.runtime.yml"

tg_token="${ALERT_TG_BOT_TOKEN:-000000000:placeholder}"
tg_chat_id="${ALERT_TG_CHAT_ID:-1871908422}"
webhook_url="${ALERT_WEBHOOK_URL:-http://127.0.0.1:19093/alert}"

rendered="$(cat "${TEMPLATE_PATH}")"
rendered="${rendered//'${ALERT_TG_BOT_TOKEN}'/${tg_token}}"
rendered="${rendered//'${ALERT_TG_CHAT_ID}'/${tg_chat_id}}"
rendered="${rendered//'${ALERT_WEBHOOK_URL}'/${webhook_url}}"

printf "%s\n" "${rendered}" > "${RUNTIME_PATH}"
echo "[alertmanager-config] rendered: ${RUNTIME_PATH}"
