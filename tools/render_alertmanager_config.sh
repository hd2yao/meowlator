#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE_PATH="${ROOT_DIR}/infra/monitoring/alertmanager.yml"
RUNTIME_PATH="${ROOT_DIR}/infra/monitoring/alertmanager.runtime.yml"

had_tg_token=0
had_tg_chat_id=0
had_webhook_url=0
orig_tg_token=""
orig_tg_chat_id=""
orig_webhook_url=""

if [ "${ALERT_TG_BOT_TOKEN+x}" = "x" ]; then
  had_tg_token=1
  orig_tg_token="${ALERT_TG_BOT_TOKEN}"
fi
if [ "${ALERT_TG_CHAT_ID+x}" = "x" ]; then
  had_tg_chat_id=1
  orig_tg_chat_id="${ALERT_TG_CHAT_ID}"
fi
if [ "${ALERT_WEBHOOK_URL+x}" = "x" ]; then
  had_webhook_url=1
  orig_webhook_url="${ALERT_WEBHOOK_URL}"
fi

for env_file in "${ROOT_DIR}/.env.alerting.local" "${ROOT_DIR}/.env.alerting"; do
  if [ -f "${env_file}" ]; then
    set -a
    # shellcheck disable=SC1090
    . "${env_file}"
    set +a
    break
  fi
done

if [ "${had_tg_token}" -eq 1 ]; then
  ALERT_TG_BOT_TOKEN="${orig_tg_token}"
fi
if [ "${had_tg_chat_id}" -eq 1 ]; then
  ALERT_TG_CHAT_ID="${orig_tg_chat_id}"
fi
if [ "${had_webhook_url}" -eq 1 ]; then
  ALERT_WEBHOOK_URL="${orig_webhook_url}"
fi
tg_token="${ALERT_TG_BOT_TOKEN:-000000000:placeholder}"
tg_chat_id="${ALERT_TG_CHAT_ID:-1871908422}"
webhook_url="${ALERT_WEBHOOK_URL:-http://127.0.0.1:19093/alert}"

rendered="$(cat "${TEMPLATE_PATH}")"
rendered="${rendered//'${ALERT_TG_BOT_TOKEN}'/${tg_token}}"
rendered="${rendered//'${ALERT_TG_CHAT_ID}'/${tg_chat_id}}"
rendered="${rendered//'${ALERT_WEBHOOK_URL}'/${webhook_url}}"

printf "%s\n" "${rendered}" > "${RUNTIME_PATH}"
echo "[alertmanager-config] rendered: ${RUNTIME_PATH}"
