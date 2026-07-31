# sshw

![GitHub](https://img.shields.io/github/license/yinheli/sshw) ![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/yinheli/sshw)

ssh client wrapper for automatic login.

![usage](./assets/sshw-demo.gif)

## install

use `go get`

```
go install github.com/yinheli/sshw/cmd/sshw@latest
```

or download binary from [releases](//github.com/yinheli/sshw/releases).

## sftp

Use the same host configuration and selector to start an interactive SFTP session:

```bash
sshw -f
```

You can also connect directly by alias or load hosts from `~/.ssh/config`:

```bash
sshw -f dev
sshw -s -f
sshw -s -f dev
```

The SFTP prompt supports:

```text
pwd, lpwd, ls, lls, cd, lcd
get [-f] [-r] <remote> [local]
reget [-f] <remote> [local]
put [-f] [-r] <local> [remote]
reput [-f] <local> [remote]
mkdir, rm, rmdir, rename
help, exit
```

The interactive prompt supports command history with the up/down arrow keys,
command-name completion with Tab, and live upload/download progress.

Use `get -r` or `put -r` to transfer directory trees. Symbolic links are
skipped, empty directories are preserved, and each regular file uses the same
safe commit behavior as a single-file transfer.

Interrupted transfers preserve a deterministic `.sshw.part` file and its
`.meta` record. Resume them with `reget` or `reput`. Before appending any data,
sshw verifies the source size, modification time, source fingerprint, and the
existing partial-file prefix. A changed source or mismatched partial file is
rejected instead of producing a corrupted result.

Downloads and uploads use temporary files and only rename them into place after
the transfer completes. Existing files are preserved unless `-f` is supplied to
the `get` or `put` command. Remote overwrites use the server's atomic POSIX
rename extension when available. On servers without that extension, sshw backs
up the original before committing the uploaded file and reports the backup path
if the operation cannot be completed or rolled back.

## config

config file load in following order:

- `~/.sshw`
- `~/.sshw.yml`
- `~/.sshw.yaml`
- `./.sshw`
- `./.sshw.yml`
- `./.sshw.yaml`

config example:

<!-- prettier-ignore -->
```yaml
- { name: dev server fully configured, user: appuser, host: 192.168.8.35, port: 22, password: 123456 }
- { name: dev server with key path, user: appuser, host: 192.168.8.35, port: 22, keypath: /root/.ssh/id_rsa }
- { name: dev server with passphrase key, user: appuser, host: 192.168.8.35, port: 22, keypath: /root/.ssh/id_rsa, passphrase: abcdefghijklmn}
- { name: dev server without port, user: appuser, host: 192.168.8.35 }
- { name: dev server without user, host: 192.168.8.35 }
- { name: dev server without password, host: 192.168.8.35 }
- { name: ⚡️ server with emoji name, host: 192.168.8.35 }
- { name: server with alias, alias: dev, host: 192.168.8.35 }
- name: server with jump
  user: appuser
  host: 192.168.8.35
  port: 22
  password: 123456
  jump:
  - user: appuser
    host: 192.168.8.36
    port: 2222


# server group 1
- name: server group 1
  children:
  - { name: server 1, user: root, host: 192.168.1.2 }
  - { name: server 2, user: root, host: 192.168.1.3 }
  - { name: server 3, user: root, host: 192.168.1.4 }

# server group 2
- name: server group 2
  children:
  - { name: server 1, user: root, host: 192.168.2.2 }
  - { name: server 2, user: root, host: 192.168.3.3 }
  - { name: server 3, user: root, host: 192.168.4.4 }
```

# callback

<!-- prettier-ignore -->
```yaml
- name: dev server fully configured
  user: appuser
  host: 192.168.8.35
  port: 22
  password: 123456
  callback-shells:
    - { cmd: 2 }
    - { delay: 1500, cmd: 0 }
    - { cmd: "echo 1" }
```
