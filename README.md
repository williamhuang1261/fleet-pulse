# fleet-pulse

A single-binary Go agent that polls a bare-metal Linux host's own vitals
(CPU, memory, disk, network) and serves them on a Prometheus-compatible
`/metrics` endpoint, deployed via a systemd unit with no container runtime,
and gating that endpoint with a locally decrypted (age-encrypted) bearer
token rather than a cloud secrets manager.

Under active development — see `deploy/`, `internal/`, and commit history
for progress. A full write-up lands in this README once the agent, its
systemd unit, and its tests are in place.
