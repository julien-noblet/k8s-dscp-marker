# Examples

These are generic, illustrative examples. Replace the container image and
command with your actual workload; only the `metadata.annotations` field
matters to k8s-dscp-marker.

- [`voice-ef.yaml`](voice-ef.yaml): a latency-sensitive workload (e.g. VoIP,
  real-time DNS) marked `EF` (Expedited Forwarding).
- [`control-plane-cs6.yaml`](control-plane-cs6.yaml): cluster control-plane /
  routing traffic marked `CS6`.
- [`database-af31.yaml`](database-af31.yaml): a database or cache workload
  marked `AF31`.
- [`web-af21.yaml`](web-af21.yaml): a web/API frontend marked `AF21`.
- [`bulk-backup-cs1.yaml`](bulk-backup-cs1.yaml): a bulk/background transfer
  (e.g. backups to object storage) marked `CS1`, the lowest-priority class
  in this set.

Apply the annotation to an existing Deployment instead of a whole manifest
with:

```sh
kubectl patch deployment my-app -p \
  '{"spec":{"template":{"metadata":{"annotations":{"dscp-marker.io/class":"AF21"}}}}}'
```
