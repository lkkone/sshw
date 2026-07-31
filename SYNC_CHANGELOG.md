# Multi-device sync release line

Synchronization-enhanced builds use `sync-vX.Y.Z` tags and are published as
GitHub prereleases. This keeps them separate from the main sshw `vX.Y.Z`
release line and prevents them from replacing the main project's latest
release.

## Unreleased

### Deployment

- Deploy by copying `.env.example` to `.env`, optionally editing it, and
  running `docker compose up -d --build`
- Use host port `9110` by default
- Generate and persist a random administrator password and encryption key on
  first startup when no `.env` overrides are supplied
- Keep the SQLite database and generated secrets in the stable
  `sshw-config-data` Docker volume
- Remove generated Archify runtime-diagram artifacts from the deployable branch

## sync-v1.0.1

Fixes local configuration drift detection.

### Fixed

- Verify the current target file's SHA-256 before sending a cached ETag
- Download and restore the published configuration when the local file was
  edited, deleted, or no longer matches the last synchronized content
- Report `local configuration changed; sync required` from `sshw sync status`
  when the remote version is unchanged but the local file has drifted

## sync-v1.0.0

First multi-device configuration synchronization release.

### Highlights

- Self-hosted configuration center with a visual host and group editor
- Import of existing `~/.sshw` YAML files, with replace and append modes
- Draft validation, generated YAML preview, publishing, and version history
- Per-device read-only synchronization tokens with independent revocation
- `sshw sync init`, `sshw sync`, `sshw sync status`, `--dry-run`, and `--force`
- Atomic local replacement, automatic backups, checksum validation, and
  backup retention
- Encrypted SQLite storage for configuration secrets and one-way token hashes
- Chinese and English Web interfaces
- Docker Compose deployment with persistent storage and health checks

### Compatibility and safety

- Existing sshw configuration files remain supported
- Jump-host names are optional, matching the behavior of the original CLI
- Plain HTTP synchronization is rejected except when explicitly enabled for
  localhost development
- Reinitializing synchronization clears stale version state so changing the
  target path always downloads the configuration again
- The configuration center is optional; SSH and SFTP behavior remains
  available without it

### Deployment

Use the included `docker-compose.yml`, `.env.example`, and `Dockerfile` to run
the configuration center. For Internet-facing deployments, place it behind an
HTTPS reverse proxy and back up the Docker volume together with the master key.
