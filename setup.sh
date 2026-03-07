#!/bin/bash
set -euo pipefail

# setup.sh — Turn a blank exe.dev VM into a persistent agent
#
# Usage: ./setup.sh [owner-email]
#
# If owner-email is not provided, attempts to detect it from exe.dev.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
AGENT_DIR="${HOME}/.agent"
VM_NAME="$(hostname)"

echo "=== Persistent Agent Setup ==="
echo "VM: ${VM_NAME}"
echo "Agent dir: ${AGENT_DIR}"
echo ""

# --- Detect owner email ---
OWNER_EMAIL="${1:-}"
if [[ -z "${OWNER_EMAIL}" ]]; then
    # Try to detect from exe.dev
    OWNER_EMAIL=$(curl -sf http://169.254.169.254/gateway/email/whoami 2>/dev/null || true)
    if [[ -z "${OWNER_EMAIL}" ]]; then
        echo "ERROR: Could not detect owner email."
        echo "Usage: ./setup.sh your@email.com"
        exit 1
    fi
fi
echo "Owner: ${OWNER_EMAIL}"
echo ""

# --- Create directory structure ---
echo "Creating directory structure..."
mkdir -p "${AGENT_DIR}/memory/daily"
mkdir -p "${AGENT_DIR}/memory/weekly"
mkdir -p "${AGENT_DIR}/bin"
mkdir -p "${AGENT_DIR}/prompts"
mkdir -p "${AGENT_DIR}/logs"

# --- Write owner email ---
echo "${OWNER_EMAIL}" > "${AGENT_DIR}/owner-email"

# --- Copy files ---
echo "Installing files..."

# bin
cp "${SCRIPT_DIR}/bin/run-prompt.sh" "${AGENT_DIR}/bin/"
chmod +x "${AGENT_DIR}/bin/run-prompt.sh"

# prompts
cp "${SCRIPT_DIR}/prompts/"*.md "${AGENT_DIR}/prompts/"

# Identity — only if not already present (don't overwrite customizations)
if [[ ! -f "${AGENT_DIR}/identity.md" ]]; then
    sed -e "s/{{VM_NAME}}/${VM_NAME}/g" \
        -e "s/{{OWNER_EMAIL}}/${OWNER_EMAIL}/g" \
        "${SCRIPT_DIR}/identity.md" > "${AGENT_DIR}/identity.md"
    echo "  Created identity.md"
else
    echo "  identity.md already exists, skipping"
fi

# LONGTERM.md — only if not already present
if [[ ! -f "${AGENT_DIR}/memory/LONGTERM.md" ]]; then
    cat > "${AGENT_DIR}/memory/LONGTERM.md" << 'EOF'
# Long-Term Memory

This file contains durable knowledge that persists across days and weeks.
The agent appends during evening/weekly consolidation. The owner may edit anytime.

## VM Setup

- Persistent agent installed on: $(date +%Y-%m-%d)
- VM name: {{VM_NAME}}
EOF
    sed -i "s/{{VM_NAME}}/${VM_NAME}/g" "${AGENT_DIR}/memory/LONGTERM.md"
    sed -i "s/\$(date +%Y-%m-%d)/$(date +%Y-%m-%d)/g" "${AGENT_DIR}/memory/LONGTERM.md"
    echo "  Created LONGTERM.md"
else
    echo "  LONGTERM.md already exists, skipping"
fi

# AGENTS.md — replace Shelley's root guidance
AGENTS_TARGET="${HOME}/.config/shelley/AGENTS.md"
if [[ -d "${HOME}/.config/shelley" ]]; then
    # Substitute the owner email placeholder in AGENTS.md
    sed "s/OWNER_EMAIL/${OWNER_EMAIL}/g" "${SCRIPT_DIR}/AGENTS.md" > "${AGENTS_TARGET}"
    echo "  Updated ~/.config/shelley/AGENTS.md"
else
    echo "  WARNING: ~/.config/shelley/ not found. Is Shelley installed?"
fi

# --- Install systemd units ---
echo ""
echo "Installing systemd timers..."

for unit in "${SCRIPT_DIR}/systemd/"*; do
    NAME=$(basename "${unit}")
    sudo cp "${unit}" "/etc/systemd/system/${NAME}"
    echo "  Installed ${NAME}"
done

sudo systemctl daemon-reload

# Enable and start all timers
for timer in agent-morning agent-evening agent-health agent-weekly agent-curiosity; do
    sudo systemctl enable --now "${timer}.timer" 2>/dev/null
    echo "  Enabled ${timer}.timer"
done

# --- Seed today's daily log ---
TODAY=$(date +%Y-%m-%d)
DAILY_FILE="${AGENT_DIR}/memory/daily/${TODAY}.md"
if [[ ! -f "${DAILY_FILE}" ]]; then
    cat > "${DAILY_FILE}" << EOF
# Daily Log — ${TODAY}

## Setup — $(date +%H:%M)

- Persistent agent installed
- Owner: ${OWNER_EMAIL}
- Timers: morning (07:00), evening (22:00), health (6h), weekly (Sun 22:30)
EOF
    echo ""
    echo "Seeded today's daily log"
fi

# --- Quick test ---
echo ""
echo "Running quick test..."
if command -v shelley &>/dev/null; then
    RESULT=$(shelley client chat -p "You are being tested. Read ~/.agent/identity.md and confirm you can see it. Reply in one sentence." 2>&1)
    CONV_ID=$(echo "${RESULT}" | jq -r '.conversation_id // "failed"' 2>/dev/null || echo "failed")
    if [[ "${CONV_ID}" != "failed" ]]; then
        echo "Test conversation started: ${CONV_ID}"
        echo "Waiting for completion..."
        timeout 60 shelley client read -wait "${CONV_ID}" 2>/dev/null | jq -r 'select(.type=="agent") | .text' 2>/dev/null | tail -1
    else
        echo "WARNING: shelley test failed. Is shelley running?"
    fi
else
    echo "WARNING: shelley not found in PATH"
fi

# --- Summary ---
echo ""
echo "=== Setup Complete ==="
echo ""
echo "Directory:    ${AGENT_DIR}"
echo "Owner:        ${OWNER_EMAIL}"
echo "Identity:     ${AGENT_DIR}/identity.md"
echo "Long-term:    ${AGENT_DIR}/memory/LONGTERM.md"
echo "Daily logs:   ${AGENT_DIR}/memory/daily/"
echo "Weekly logs:  ${AGENT_DIR}/memory/weekly/"
echo "Run log:      ${AGENT_DIR}/logs/runs.log"
echo ""
echo "Timers:"
systemctl list-timers 'agent-*' --no-pager 2>/dev/null || true
echo ""
echo "To customize:"
echo "  Edit identity:     nano ${AGENT_DIR}/identity.md"
echo "  Edit long-term:    nano ${AGENT_DIR}/memory/LONGTERM.md"
echo "  Edit schedules:    sudo systemctl edit agent-morning.timer"
echo "  View runs:         cat ${AGENT_DIR}/logs/runs.log"
echo "  Manual trigger:    ${AGENT_DIR}/bin/run-prompt.sh ${AGENT_DIR}/prompts/morning.md"
echo "  Uninstall:         sudo systemctl disable --now agent-{morning,evening,health,weekly}.timer"
