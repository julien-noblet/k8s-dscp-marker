# k8s-dscp-marker

A tiny, annotation-driven Kubernetes DaemonSet that marks the DSCP field of
a Pod's outgoing traffic, so that annotation-aware upstream network
equipment (switches, routers) can apply per-class QoS policies without any
cluster-side CRD, webhook, or service mesh.

## Why

Most CNIs (including Cilium, as of this writing) have no built-in way to set
the DSCP bits of a Pod's egress traffic from a simple annotation. If your
network already has QoS configured (e.g. WRR queues classified by DSCP on a
managed switch), Pod traffic typically leaves the cluster unmarked and falls
into whatever default class your network assigns — which can throttle it far
below the link's actual capacity.

k8s-dscp-marker closes that gap with the smallest possible mechanism: no
CRD, no admission webhook, no mesh sidecar — just a Pod annotation and a
dedicated iptables chain per node.

## How it works

- One DaemonSet Pod per node, running with `hostNetwork: true` and the
  `NET_ADMIN` capability.
- Each instance watches only the Pods scheduled on its own node
  (`spec.nodeName` field selector) — no cluster-wide fan-out.
- For every Pod carrying the `dscp-marker.io/class` annotation, it adds a
  rule to a dedicated `DSCP_MARKER` chain in the `mangle` table:
  ```
  iptables -t mangle -A DSCP_MARKER -s <pod-ip>/32 -j DSCP --set-dscp-class <class>
  ```
- `POSTROUTING` jumps to `DSCP_MARKER` once; the chain is flushed and
  rebuilt on every reconciliation. **No other chain or rule is ever
  touched** — it coexists safely with CNI-managed iptables rules (e.g.
  Cilium's own chains).
- Reconciliation runs on every Pod add/update/delete event (debounced) and
  is replayed periodically by the underlying `client-go` informer resync,
  so it self-heals even if an event is ever missed.

## Installation

```sh
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/daemonset.yaml
```

Pin the image to a released tag in `deploy/daemonset.yaml` before deploying
to production (see [Releases](../../releases)).

## Usage

Annotate any Pod (directly, or via a Deployment/CronJob/StatefulSet
template) with the DSCP class you want applied to its egress traffic:

```yaml
metadata:
  annotations:
    dscp-marker.io/class: "EF"
```

Accepted values are the DSCP class names understood by the iptables `DSCP`
module: `default`, `CS0`-`CS7`, `AF11`-`AF43`, `EF`. See [`examples/`](examples)
for complete manifests covering a few common traffic classes.

An invalid class name is logged and skipped; it never blocks the other
rules from being applied.

## Configuration

Environment variables read by the controller (all optional except
`NODE_NAME`, which is normally set via the Downward API in the DaemonSet
manifest):

| Variable         | Default               | Description                                   |
|------------------|-----------------------|------------------------------------------------|
| `NODE_NAME`      | *(required)*          | Node this instance watches Pods on.             |
| `ANNOTATION_KEY` | `dscp-marker.io/class`| Annotation key read on each Pod.                |
| `CHAIN_NAME`     | `DSCP_MARKER`         | Name of the dedicated iptables mangle chain.    |

Run `k8s-dscp-marker -kubeconfig=~/.kube/config` for local development
outside a cluster.

## Limitations

- IPv4 `mangle`/`iptables` only; no IPv6 or `nftables` support yet.
- The chain is flushed and refilled on every reconciliation, which is not
  atomic: a packet can cross the chain mid-update and briefly miss its
  mark. Acceptable for most use cases; if you need atomicity, consider
  `iptables-restore --table=mangle --noflush` instead.
- Marks egress traffic leaving the Pod's node only; it does not affect
  traffic that stays entirely within the same node's overlay/tunnel, nor
  does it mark return traffic.
- No validation beyond the static DSCP class name list — it does not check
  that your upstream network actually honors the class you chose.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
