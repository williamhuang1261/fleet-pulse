# Runbook

fleet-pulse fails loudly and exits non-zero rather than starting in a broken
state. Every failure below was reproduced locally; the log line is pasted
verbatim, not invented.

## Config file missing

```
$ fleet-pulse --config /nonexistent/config.yaml
config: open /nonexistent/config.yaml: no such file or directory
```

**Fix:** check the `--config` path. If the config was meant to come from
`/etc/fleet-pulse/config.yaml` (the systemd deploy's default), confirm
`deploy/install.sh` actually ran and that file exists.

## Config file has invalid YAML

```
$ fleet-pulse --config config.yaml
config: yaml: line 1: did not find expected ',' or ']'
```

**Fix:** the message names the line; check that line for a stray bracket,
missing quote, or bad indentation. `interval` must be a Go duration string
(`5s`, `1m`) and `listen` a quoted address (`":9090"`).

## Secrets file has the wrong permissions

```
$ fleet-pulse --listen :9090 --secrets-file secrets.txt.age --identity-file key.txt
secrets: open /path/to/secrets.txt.age: permission denied
```

**Fix:** the process's user (under systemd, the `fleet-pulse` system user
`deploy/install.sh` creates) needs read access to the secrets file. Check
`ls -l` on the file and that it's owned by or readable by that user.

## `--secrets-file` set without `--identity-file`

```
$ fleet-pulse --listen :9090 --secrets-file secrets.txt.age
--secrets-file requires --identity-file
```

**Fix:** decrypting the secrets file needs the matching age identity (private
key). Pass `--identity-file` pointing at the `key.txt` produced by
`age-keygen` when the secrets file was encrypted.

## Wrong identity for the secrets file

If `--identity-file` points at an identity that doesn't match the recipient
the secrets file was encrypted for, decryption itself fails:

```
secrets: decrypting /path/to/secrets.txt.age: identity did not match any of the recipients: incorrect identity for recipient block
```

**Fix:** re-encrypt the secrets file for the identity actually being used
(`age -r <recipient from age-keygen -o key.txt> -o secrets.txt.age`), or
point `--identity-file` at the correct key.

## Listen address already in use

```
$ fleet-pulse --listen :9090
2026/09/01 15:01:26 fleet-pulse: /metrics is open (no --secrets-file set)
2026/09/01 15:01:26 fleet-pulse: serving /metrics on :9090
listen: listen tcp :9090: bind: address already in use
```

**Fix:** something else (often a previous fleet-pulse instance that didn't
exit cleanly) already owns the port. `lsof -i :9090` (or `ss -ltnp` on
Linux) to find it; under systemd, `systemctl status fleet-pulse` first --
a stuck old instance usually means the service needs a `systemctl restart`,
not a second manual instance started alongside it.

## Checking a running systemd deployment

```
systemctl status fleet-pulse       # is it up, and since when
journalctl -u fleet-pulse -n 50    # the log lines above show up here
curl localhost:9090/metrics        # confirm it's actually serving
```

If the service is enabled but not running after a fresh `install.sh`,
`journalctl -u fleet-pulse` will contain one of the failure modes above --
the ones above are exactly the strings to grep for.
