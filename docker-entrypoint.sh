#!/bin/sh
set -eu

secret_dir="${SSHW_SECRET_DIR:-/data/secrets}"
mkdir -p "${secret_dir}"
chmod 700 "${secret_dir}"
umask 077

generate_secret() {
  output_path="$1"
  byte_count="$2"
  prefix="$3"

  if [ ! -s "${output_path}" ]; then
    {
      printf '%s' "${prefix}"
      head -c "${byte_count}" /dev/urandom | base64 | tr -d '\n'
      printf '\n'
    } >"${output_path}"
    chmod 600 "${output_path}"
    return 0
  fi

  return 1
}

if [ -z "${SSHW_MASTER_KEY:-}" ] && [ -z "${SSHW_MASTER_KEY_FILE:-}" ]; then
  SSHW_MASTER_KEY_FILE="${secret_dir}/master-key"
  export SSHW_MASTER_KEY_FILE
  generate_secret "${SSHW_MASTER_KEY_FILE}" 32 "" || true
fi

if [ -z "${SSHW_ADMIN_PASSWORD:-}" ] && [ -z "${SSHW_ADMIN_PASSWORD_FILE:-}" ]; then
  SSHW_ADMIN_PASSWORD_FILE="${secret_dir}/admin-password"
  export SSHW_ADMIN_PASSWORD_FILE
  if generate_secret "${SSHW_ADMIN_PASSWORD_FILE}" 18 "sshw_"; then
    echo "============================================================"
    echo "sshw configuration center initialized"
    echo "username: ${SSHW_ADMIN_USERNAME:-admin}"
    echo "password: $(tr -d '\r\n' <"${SSHW_ADMIN_PASSWORD_FILE}")"
    echo "Save this password. It is also persisted in the data volume."
    echo "============================================================"
  fi
fi

exec "$@"
