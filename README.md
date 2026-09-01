# fleet-pulse

A single-binary Go agent that polls a bare-metal Linux host's own vitals
(CPU, memory, disk, network) and serves them on a Prometheus-compatible
`/metrics` endpoint, deployed via a systemd unit with no container runtime,
and gating that endpoint with a locally decrypted (age-encrypted) bearer
token rather than a cloud secrets manager.

## What it does

- Polls host CPU, memory, disk and network vitals via [`gopsutil`](https://github.com/shirou/gopsutil).
- Prints them as JSON to stdout on an interval (`--once` for a single reading), **or**
- Serves them as a live Prometheus exporter on `/metrics` (`--listen`), reading
  the host fresh on every scrape rather than a background-polled cache.
- Optionally gates `/metrics` behind a bearer token that is itself decrypted
  from a local [age](https://age-encryption.org/)-encrypted file at startup
  — no plaintext secret on disk, no cloud KMS.
- Ships as a static Linux binary with a systemd unit and an idempotent
  install script.

## Why

Most host-metrics agents either assume a container runtime (a Docker/K8s
sidecar) or a cloud provider's own agent. fleet-pulse targets the other
case: a bare-metal Linux box running systemd, where the simplest thing that
works is a single static binary and a `.service` file, no image to build,
no registry to push to, no orchestrator to keep healthy.

## Stack

- **Go** — a static binary needs no runtime installed on the target host.
- **gopsutil** — cross-platform host metrics without hand-parsing `/proc`
  formats that vary by kernel version.
- **prometheus/client_golang** — the de facto standard `/metrics` exposition
  format.
- **filippo.io/age** — modern, minimal, single-purpose encryption (by the
  maintainer of Go's own crypto libraries), used here as SOPS-style
  file-based secret-at-rest decryption. See "Engineering notes" below for
  why this is age and not literally SOPS.
- **systemd** — the target host's own init system; no container runtime.

## Run it

```
go run . --once                              # one JSON snapshot to stdout
go run . --interval 10s                      # JSON snapshots on a loop
go run . --listen :9090                      # serve /metrics, no auth
go run . --config config.example.yaml        # interval/listen from a file
```

Sample `--listen` output (`curl localhost:9090/metrics`, captured from a
real run on 2026-09-01):

```
# HELP fleet_pulse_cpu_percent Current CPU utilization, in percent.
# TYPE fleet_pulse_cpu_percent gauge
fleet_pulse_cpu_percent 9.561752989960317
# HELP fleet_pulse_disk_used_percent Current disk utilization at the collector's mount point, in percent.
# TYPE fleet_pulse_disk_used_percent gauge
fleet_pulse_disk_used_percent 70.65491528115385
# HELP fleet_pulse_mem_used_percent Current memory utilization, in percent.
# TYPE fleet_pulse_mem_used_percent gauge
fleet_pulse_mem_used_percent 76.39760971069336
# HELP fleet_pulse_net_bytes_recv_total Cumulative bytes received over all network interfaces since boot.
# TYPE fleet_pulse_net_bytes_recv_total counter
fleet_pulse_net_bytes_recv_total 7.2250011609e+10
# HELP fleet_pulse_net_bytes_sent_total Cumulative bytes sent over all network interfaces since boot.
# TYPE fleet_pulse_net_bytes_sent_total counter
fleet_pulse_net_bytes_sent_total 2.1359705917e+10
```

### Gating `/metrics` with a bearer token

```
age-keygen -o key.txt                                   # generates an identity + prints its public key
echo -n "my-bearer-token" | age -r <public key> -o secrets.txt.age
go run . --listen :9090 --secrets-file secrets.txt.age --identity-file key.txt
curl -H "Authorization: Bearer my-bearer-token" localhost:9090/metrics
```

Leaving `--secrets-file` unset keeps `/metrics` open, a documented default
for a trusted internal network, not an oversight.

### Deploying with systemd

```
make build            # cross-compiles a static linux/amd64 binary
sudo ./deploy/install.sh
systemctl status fleet-pulse
```

See [`docs/runbook.md`](docs/runbook.md) for what each real startup/runtime
failure looks like in the logs.

## Testing

```
go test ./...
```

7 test functions (16 assertions counting subtests) across three packages:
table-driven collector tests (including a canceled-context case), a live
`httptest` round trip through the Prometheus exporter, and secrets-package
tests covering identity loading, correct/incorrect decryption, and the
bearer-token HTTP middleware's four auth outcomes. `go vet` and `gofmt -l`
are enforced in CI (`.github/workflows/ci.yml`) on every push and PR.

## Engineering notes

**Why age instead of SOPS.** SOPS is the more common tool for this job, but
its dependency tree and typical workflow assume a cloud KMS backend (AWS
KMS, GCP KMS, Vault), exactly what this project is deliberately avoiding.
age is a minimal, single-purpose primitive: one keypair, one encrypted file,
no cloud dependency. It's a deliberate, defensible substitution for the
specific need here (a single small secret, decrypted locally), documented
as such rather than claimed as literal SOPS support.

**Why the exporter reads live instead of caching on an interval.** The
original plan called for a background-polled cache refreshed on `interval`.
Implementing it, a `prometheus.Collector` that takes one real reading per
scrape turned out simpler (no goroutine, no staleness window to reason
about) and is the standard pattern for a collector this cheap to read. The
interval setting still governs stdout mode, where a poll loop is the whole
point.

**What's still first-party bare-metal, not containerized.** No Docker, no
Kubernetes manifests, no cloud SDK imports anywhere in this repo, the
static binary plus systemd unit is the only deploy path, on purpose.

## What's deliberately not here

- Metric *set* configurability, the exporter always emits all five metrics;
  the config file only tunes interval/listen, not which metrics are on.
- A live end-to-end test of `deploy/install.sh`, it needs root on a real
  Linux host with systemd, which this dev machine doesn't have. It's
  syntax-checked (`bash -n`) and reviewed by hand against `systemd.service(5)`;
  see `docs/runbook.md` and the build plan's Step 5 log for specifics.
- Log shipping, alerting rules, or a Grafana dashboard, this project stops
  at exposing metrics for something else to scrape.
