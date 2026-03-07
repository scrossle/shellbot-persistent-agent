#!/bin/bash
set -euo pipefail

# run-prompt.sh — run a prompt file through shelley
# Usage: run-prompt.sh <prompt-file> [model]
#
# Reads the prompt file, substitutes date variables, and fires a shelley conversation.
# Logs the conversation ID to ~/.agent/logs/runs.log

PROMPT_FILE="${1:?Usage: run-prompt.sh <prompt-file> [model]}"
MODEL="${2:-claude-haiku-4.5}"
AGENT_DIR="${HOME}/.agent"
LOG_DIR="${AGENT_DIR}/logs"
DAILY_DIR="${AGENT_DIR}/memory/daily"

mkdir -p "${LOG_DIR}" "${DAILY_DIR}"

# Date variables for substitution
TODAY=$(date +%Y-%m-%d)
YESTERDAY=$(date -d 'yesterday' +%Y-%m-%d)
WEEK=$(date +%Y-W%V)
OWNER_EMAIL=$(cat "${AGENT_DIR}/owner-email" 2>/dev/null || echo "")
VM_NAME=$(hostname)

# Read and substitute the prompt
if [[ ! -f "${PROMPT_FILE}" ]]; then
    echo "ERROR: Prompt file not found: ${PROMPT_FILE}" >&2
    exit 1
fi

PROMPT=$(cat "${PROMPT_FILE}")
PROMPT="${PROMPT//\{\{TODAY\}\}/${TODAY}}"
PROMPT="${PROMPT//\{\{YESTERDAY\}\}/${YESTERDAY}}"
PROMPT="${PROMPT//\{\{WEEK\}\}/${WEEK}}"
PROMPT="${PROMPT//\{\{OWNER_EMAIL\}\}/${OWNER_EMAIL}}"
PROMPT="${PROMPT//\{\{VM_NAME\}\}/${VM_NAME}}"

# Build shelley args
ARGS=(client chat -p "${PROMPT}")
if [[ -n "${MODEL}" ]]; then
    ARGS+=(-model "${MODEL}")
fi

# Run
TIMESTAMP=$(date -Iseconds)
RESULT=$(shelley "${ARGS[@]}" 2>&1)
CONV_ID=$(echo "${RESULT}" | jq -r '.conversation_id // "unknown"' 2>/dev/null || echo "unknown")

# Log the run
echo "${TIMESTAMP} prompt=$(basename "${PROMPT_FILE}") conv=${CONV_ID}" >> "${LOG_DIR}/runs.log"

# Wait for completion (timeout 10 minutes)
if [[ "${CONV_ID}" != "unknown" ]]; then
    timeout 600 shelley client read -wait "${CONV_ID}" > /dev/null 2>&1 || true
fi

echo "${CONV_ID}"
