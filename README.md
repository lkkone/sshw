# sshw

[![Release](https://img.shields.io/github/v/release/lkkone/sshw)](https://github.com/lkkone/sshw/releases/latest)
[![Build](https://github.com/lkkone/sshw/actions/workflows/build.yml/badge.svg)](https://github.com/lkkone/sshw/actions/workflows/build.yml)
[![License](https://img.shields.io/github/license/lkkone/sshw)](LICENSE)

`sshw` is an interactive SSH and SFTP connection manager for the terminal.
Keep hosts, groups, aliases, credentials, and jump hosts in one place, then
connect without repeatedly typing long SSH commands.

The same host selector and configuration are shared by both modes:

```bash
sshw          # open an SSH session
sshw -f       # open an interactive SFTP session
```

## Highlights

- Interactive host selector with keyboard navigation and search
- Direct connection by a short alias
- Nested host groups and jump hosts
- Password, private-key, SSH agent, and keyboard-interactive authentication
- Import from `~/.ssh/config`
- Interactive SFTP with progress, history, and Tab completion
- Recursive and resumable SFTP transfers
- Safe overwrite handling that protects the existing destination file
- Optional self-hosted configuration center for one-command device sync
- Linux, macOS, Windows, FreeBSD, OpenBSD, and Solaris builds

## Installation

### Download a release

The recommended method is to download the package for your system from the
[latest release](https://github.com/lkkone/sshw/releases/latest).

Choose the archive that matches your operating system and CPU:

| System | Common package |
| --- | --- |
| macOS on Apple Silicon | `sshw_*_darwin_arm64.tar.gz` |
| macOS on Intel | `sshw_*_darwin_amd64.tar.gz` |
| Linux x86-64 | `sshw_*_linux_amd64.tar.gz` |
| Linux ARM64 | `sshw_*_linux_arm64.tar.gz` |
| Windows x86-64 | `sshw_*_windows_amd64.zip` |

On macOS or Linux, extract the archive and place the binary somewhere in your
`PATH`:

```bash
tar -xzf sshw_*.tar.gz
sudo install sshw /usr/local/bin/sshw
sshw -version
```

Windows users can extract `sshw.exe` from the ZIP archive and add its directory
to `PATH`.

Each release also contains `sshw_*_checksums.txt` for verifying downloads.

### Build from source

Building from source requires Go 1.25 or newer:

```bash
git clone https://github.com/lkkone/sshw.git
cd sshw
go build -o sshw ./cmd/sshw
sudo install sshw /usr/local/bin/sshw
```

## Quick start

Create `~/.sshw`:

```yaml
- name: Development
  alias: dev
  host: 192.168.1.10
  user: root
  port: 22
  keypath: ~/.ssh/id_ed25519

- name: Production
  alias: prod
  host: 192.168.1.20
  user: deploy
```

Open the host selector:

```bash
sshw
```

Connect directly using an alias:

```bash
sshw dev
```

While selecting a host:

- Use the arrow keys or `j`/`k` to move
- Press Enter to select
- Press `/` to search
- Press `Ctrl+C` to cancel

## Configuration sync

sshw includes an optional self-hosted configuration center. It provides a
visual Web editor, generated YAML preview, draft and published versions,
version history, Chinese and English interfaces, and a separate read-only
token for every computer. The Web interface defaults to Simplified Chinese
and remembers the selected language.

Multi-device synchronization enhancements use a separate `sync-vX.Y.Z`
release line. These releases are published as prereleases so they do not
replace or conflict with the main sshw `vX.Y.Z` release line.

The server is the single source of truth. The CLI never uploads local changes:

```bash
sshw sync
```

The command checks the latest published version, validates its checksum and
YAML structure, backs up the current configuration, and safely replaces
`~/.sshw`. A failed request or invalid remote configuration never changes the
working local file.

### Connect a computer

Create a device token in the Web interface, then initialize the computer:

```bash
sshw sync init \
  --server https://sshw.example.com \
  --token sshw_sync_xxxxxxxxxxxxxxxxx
```

This creates `~/.sshw-sync.yaml` with permission `0600`. After that:

```text
sshw sync               Download the latest published configuration
sshw sync status        Compare the local and remote versions
sshw sync --dry-run     Download and validate without replacing ~/.sshw
sshw sync --force       Download again even when the version is unchanged
```

Plain HTTP is rejected except for localhost. For local development:

```bash
sshw sync init \
  --server http://127.0.0.1:8080 \
  --token sshw_sync_xxxxxxxxxxxxxxxxx \
  --allow-insecure
```

### Run the configuration center

Docker Compose is included in the repository:

```bash
cp .env.example .env
openssl rand -base64 32
```

Put the generated value in `.env` as `SSHW_MASTER_KEY`, choose a strong
`SSHW_ADMIN_PASSWORD`, then start the service:

```bash
docker compose up -d --build
docker compose ps
```

Open `http://127.0.0.1:8080`, sign in, add hosts or groups, save the draft, and
publish the first version. Create a token from **Sync devices** for each
computer that should receive the configuration.

To migrate an existing configuration, choose **Import configuration** and
select the current `~/.sshw` file or another YAML file. The server validates
the complete file before opening it as an unsaved draft. You can replace the
current draft or append the imported entries, review the generated YAML, and
then save and publish when ready. Importing alone never changes the saved or
published configuration.

The SQLite database is stored in the `sshw-config-data` Docker volume. Passwords,
passphrases, drafts, and published YAML are encrypted before they are written
to the database. Device tokens are stored as one-way hashes.

For an Internet-facing deployment:

- Put the service behind HTTPS using Caddy, Nginx, or another reverse proxy
- Use a randomly generated master key and strong administrator password
- Keep `.env` out of source control
- Back up the Docker volume and master key together
- Do not expose port `8080` directly when a reverse proxy is available

Changing or losing `SSHW_MASTER_KEY` makes existing encrypted data unreadable,
so store it in a secure backup or Docker secret.

## SSH usage

```text
sshw                    Select a host and connect with SSH
sshw <alias>            Connect directly by alias
sshw -s                 Select a host from ~/.ssh/config
sshw -s <alias>         Connect to an alias from ~/.ssh/config
sshw -version           Show version information
sshw -help              Show command-line options
```

## SFTP usage

Add `-f` to use the same host selection and authentication flow with SFTP:

```text
sshw -f                 Select a host and start SFTP
sshw -f <alias>         Start SFTP directly by alias
sshw -s -f              Select a host from ~/.ssh/config and start SFTP
sshw -s -f <alias>      Start SFTP for an alias from ~/.ssh/config
```

### SFTP commands

| Command | Description |
| --- | --- |
| `pwd` / `lpwd` | Show the remote/local working directory |
| `ls [path]` / `lls [path]` | List remote/local files |
| `cd <path>` / `lcd <path>` | Change the remote/local directory |
| `get [-f] <remote> [local]` | Download a file |
| `put [-f] <local> [remote]` | Upload a file |
| `get -r [-f] <remote> [local]` | Download a directory recursively |
| `put -r [-f] <local> [remote]` | Upload a directory recursively |
| `reget [-f] <remote> [local]` | Resume an interrupted download |
| `reput [-f] <local> [remote]` | Resume an interrupted upload |
| `mkdir <path>` | Create a remote directory |
| `rm <path>` | Remove a remote file |
| `rmdir <path>` | Remove an empty remote directory |
| `rename <old> <new>` | Rename a remote path |
| `help` | Show available commands |
| `exit` / `quit` | Close the SFTP session |

The SFTP prompt supports command history with the up/down arrow keys,
command-name completion with Tab, and live transfer progress. Recursive
transfers display one aggregate directory progress indicator instead of
printing a progress bar for every file.

### Resume and overwrite behavior

Interrupted transfers leave a `.sshw.part` file and a small `.meta` record.
Use `reget` or `reput` to continue the transfer. Before resuming, sshw checks
that the source and partial data still match, preventing a changed file from
being combined with stale data.

Existing destination files are not overwritten unless `-f` is supplied.
Transfers write to a temporary file first and rename it only after all data has
arrived. When a server does not support atomic POSIX rename, sshw temporarily
backs up the existing remote file and attempts an immediate rollback if the
replacement fails.

Recursive transfers merge into existing directories. The `-f` option is only
required when an individual destination file already exists.

## Configuration

sshw loads the first configuration file it finds in this order:

1. `~/.sshw`
2. `~/.sshw.yml`
3. `~/.sshw.yaml`
4. `./.sshw`
5. `./.sshw.yml`
6. `./.sshw.yaml`

### Hosts, groups, and aliases

```yaml
- name: Development
  alias: dev
  user: appuser
  host: 192.168.8.35
  port: 22
  keypath: ~/.ssh/id_ed25519

- name: Servers
  children:
    - name: Application 1
      alias: app-1
      user: root
      host: 192.168.8.41
    - name: Application 2
      alias: app-2
      user: root
      host: 192.168.8.42
```

`name` is shown in the selector. `alias` is optional and lets you connect
directly with commands such as `sshw app-1` or `sshw -f app-1`.

The defaults are `root` for `user` and `22` for `port`.

### Authentication

```yaml
- name: Private key
  host: 192.168.8.35
  user: appuser
  keypath: ~/.ssh/id_ed25519
  passphrase: optional-key-passphrase

- name: SSH agent
  host: 192.168.8.36
  user: appuser
  agentpath: /path/to/ssh-agent.sock

- name: Password
  host: 192.168.8.37
  user: appuser
  password: optional-password
```

Authentication methods are attempted in this order:

1. Configured SSH agent
2. Configured private key
3. `~/.ssh/id_rsa`
4. Configured password
5. Interactive password or keyboard prompt

For shared configuration files, prefer keys or an SSH agent instead of storing
passwords and passphrases as plain text.

### Jump hosts

```yaml
- name: Internal application
  alias: internal
  user: deploy
  host: 10.0.2.15
  jump:
    - name: Bastion
      user: jumpuser
      host: bastion.example.com
      port: 22
      keypath: ~/.ssh/id_ed25519
```

Multiple entries under `jump` create a multi-hop connection chain. Jump hosts
work for both SSH and SFTP.

### Use `~/.ssh/config`

Pass `-s` to load hosts from the standard OpenSSH configuration:

```bash
sshw -s
sshw -s my-host
sshw -s -f my-host
```

### SSH callback commands

SSH sessions can send commands immediately after login:

```yaml
- name: Application shell
  alias: app
  user: deploy
  host: 192.168.8.35
  callback-shells:
    - cmd: "cd /srv/application"
    - delay: 300
      cmd: "clear"
```

`delay` is measured in milliseconds. Callback commands apply to interactive
SSH sessions, not SFTP sessions.

## License

[MIT](LICENSE)
