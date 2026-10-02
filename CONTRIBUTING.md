# Contributing

Issues and pull requests are welcome.

## Local development

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .   # should print nothing
```

## Testing against a real cluster

```sh
go run . -kubeconfig=$HOME/.kube/config
```

Requires `iptables` and `NET_ADMIN` (run as root, or via `sudo`) since the
controller executes real iptables commands.

## Commit messages

Short, imperative subject line (e.g. "Add IPv6 support"); explain the "why"
in the body when it's not obvious from the diff.
