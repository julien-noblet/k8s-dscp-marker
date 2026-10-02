// Package mangle manages a dedicated iptables mangle chain that marks the
// DSCP field of outgoing packets based on a per-Pod rule set. It never
// touches any other chain or rule installed by the CNI (e.g. Cilium) or the
// cluster administrator: it only flushes and repopulates its own chain.
package mangle

import (
	"fmt"
	"os/exec"
	"sort"
)

// Rule marks all packets sourced from PodIP with the given DSCP Class.
type Rule struct {
	PodIP string
	Class string
}

// validClasses are the DSCP class names recognized by the iptables DSCP
// module (xt_DSCP). Validating here, rather than letting iptables reject an
// unknown name mid-run, keeps a single bad annotation from leaving the chain
// half-populated.
var validClasses = map[string]bool{
	"default": true,
	"CS0":     true, "CS1": true, "CS2": true, "CS3": true,
	"CS4": true, "CS5": true, "CS6": true, "CS7": true,
	"AF11": true, "AF12": true, "AF13": true,
	"AF21": true, "AF22": true, "AF23": true,
	"AF31": true, "AF32": true, "AF33": true,
	"AF41": true, "AF42": true, "AF43": true,
	"EF": true,
}

// IsValidClass reports whether class is a DSCP class name understood by
// iptables' DSCP module.
func IsValidClass(class string) bool {
	return validClasses[class]
}

// EnsureChain creates the chain (if missing) and makes sure POSTROUTING
// jumps to it exactly once. Safe to call repeatedly.
func EnsureChain(chain string) error {
	// Create the chain; ignore the error if it already exists.
	_ = run("-t", "mangle", "-N", chain)

	if err := run("-t", "mangle", "-C", "POSTROUTING", "-j", chain); err != nil {
		if err := run("-t", "mangle", "-I", "POSTROUTING", "1", "-j", chain); err != nil {
			return fmt.Errorf("inserting jump to %s in POSTROUTING: %w", chain, err)
		}
	}
	return nil
}

// Commands returns the ordered iptables argument lists that reconcile chain
// to contain exactly the given rules. Pure/testable: no iptables is invoked.
// Invalid rules (unknown DSCP class) are skipped and returned separately so
// the caller can log them without aborting the whole reconciliation.
func Commands(chain string, rules []Rule) (cmds [][]string, skipped []Rule) {
	// Sort for deterministic, diffable output (makes tests and logs stable).
	sorted := append([]Rule(nil), rules...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].PodIP < sorted[j].PodIP })

	cmds = append(cmds, []string{"-t", "mangle", "-F", chain})
	for _, r := range sorted {
		if !IsValidClass(r.Class) {
			skipped = append(skipped, r)
			continue
		}
		cmds = append(cmds, []string{
			"-t", "mangle", "-A", chain,
			"-s", r.PodIP + "/32",
			"-j", "DSCP", "--set-dscp-class", r.Class,
		})
	}
	return cmds, skipped
}

// Reconcile flushes chain and re-applies rules. On the first failed command
// it returns an error but still attempts the remaining commands, so a single
// bad rule cannot block the others from being applied.
//
// ponytail: flush+refill is not atomic (a packet can cross the chain mid
// update and miss a mark for a few milliseconds). Acceptable for a homelab;
// switch to `iptables-restore --table=mangle --noflush` with a dedicated
// save file if you ever need atomic updates.
func Reconcile(chain string, rules []Rule) error {
	cmds, skipped := Commands(chain, rules)
	for _, r := range skipped {
		fmt.Printf("skipping pod %s: invalid DSCP class %q\n", r.PodIP, r.Class)
	}

	var firstErr error
	for _, args := range cmds {
		if err := run(args...); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("running iptables %v: %w", args, err)
		}
	}
	return firstErr
}

func run(args ...string) error {
	out, err := exec.Command("iptables", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", out, err)
	}
	return nil
}
