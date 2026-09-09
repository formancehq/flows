package audit

import (
	"fmt"
	"path/filepath"
	"sort"
)

// Record is one operation with everything this audit can prove about it.
type Record struct {
	Operation
	Family Family `json:"family"`
	Risk   Risk   `json:"risk"`
	// BaselineCommands are the legacy fctl command paths that reach this
	// operation, sorted. Empty means the operation has no legacy precedent.
	BaselineCommands []string `json:"baselineCommands"`
	// Blockers are the blocker IDs recorded against this operation, sorted.
	Blockers []string `json:"blockers"`
}

// Totals are the counts the inventory document quotes. Every one of them is
// derived, never transcribed.
type Totals struct {
	// SpecOperations is every operation in the merged document, both majors.
	SpecOperations int `json:"specOperations"`
	// UniqueOperationIDs must equal SpecOperations.
	UniqueOperationIDs int `json:"uniqueOperationIds"`
	// V1Operations and V2Operations partition SpecOperations by major.
	V1Operations int `json:"v1Operations"`
	V2Operations int `json:"v2Operations"`
	// DeprecatedOperations is the count marked deprecated, both majors.
	DeprecatedOperations int `json:"deprecatedOperations"`

	// BaselineCommands is the executable legacy fctl `orchestration` command
	// count.
	BaselineCommands int `json:"baselineCommands"`
	// BaselineMapped and BaselineExcluded partition BaselineCommands.
	BaselineMapped   int `json:"baselineMapped"`
	BaselineExcluded int `json:"baselineExcluded"`
	// BaselineDistinctOperations is the number of distinct operations the
	// baseline reaches, across both majors.
	BaselineDistinctOperations int `json:"baselineDistinctOperations"`

	// WithBaseline is the number of operations reached by the baseline;
	// WithoutBaseline is the surface that has no legacy precedent.
	WithBaseline    int `json:"withBaseline"`
	WithoutBaseline int `json:"withoutBaseline"`
	// Blocked is the number of operations carrying at least one blocker.
	Blocked int `json:"blocked"`
	// Unblocked is SpecOperations minus Blocked: proven facts, no recorded
	// blocker. It is not an acceptance claim; the runtime gates are separate.
	Unblocked int `json:"unblocked"`

	// PaginatedOperations is the number declaring cursor + pageSize.
	PaginatedOperations int `json:"paginatedOperations"`
	// UnboundedCollections is the number returning a collection with no
	// declared way to bound it.
	UnboundedCollections int `json:"unboundedCollections"`
	// MutatingOperations is the number using a state-changing HTTP method.
	MutatingOperations int `json:"mutatingOperations"`
	// DestructiveOperations is the number that remove or terminate state.
	DestructiveOperations int `json:"destructiveOperations"`
}

// Report is the whole deterministic inventory.
type Report struct {
	// SpecDocument is the base name of the document the report was built from.
	// Only the base name is recorded so the report is identical whether it is
	// produced from the module root or from a package directory.
	SpecDocument string `json:"specDocument"`
	// ProductRevision pins the orchestration tree the source facts came from.
	ProductRevision string `json:"productRevision"`
	// BaselineRevision pins the legacy fctl tree the baseline came from.
	BaselineRevision string `json:"baselineRevision"`
	Totals           Totals `json:"totals"`
	// Operations is every operation in the document, sorted by
	// (version, operationId).
	Operations []Record `json:"operations"`
}

// Build reads the document at specPath and produces the full report.
func Build(specPath string) (*Report, error) {
	ops, err := LoadOperations(specPath)
	if err != nil {
		return nil, err
	}

	byID := Index(ops)
	if len(byID) != len(ops) {
		return nil, fmt.Errorf("document declares %d operations but only %d unique operationIds", len(ops), len(byID))
	}

	commandsByOp := map[string][]string{}
	for _, c := range Baseline {
		for _, op := range c.Ops {
			commandsByOp[op] = append(commandsByOp[op], c.Path)
		}
	}
	blockersByOp := map[string][]string{}
	for _, b := range Blockers {
		for _, op := range b.OperationIDs {
			blockersByOp[op] = append(blockersByOp[op], b.ID)
		}
	}

	report := &Report{
		SpecDocument:     filepath.Base(specPath),
		ProductRevision:  ProductRevision,
		BaselineRevision: BaselineRevision,
	}

	for _, op := range ops {
		family, ok := FamilyOf(op.OperationID)
		if !ok {
			return nil, fmt.Errorf("operation %s is not classified into a family", op.OperationID)
		}
		commands := append([]string(nil), commandsByOp[op.OperationID]...)
		sort.Strings(commands)
		blocks := append([]string(nil), blockersByOp[op.OperationID]...)
		sort.Strings(blocks)
		report.Operations = append(report.Operations, Record{
			Operation:        op,
			Family:           family,
			Risk:             RiskOf(op),
			BaselineCommands: commands,
			Blockers:         blocks,
		})
	}

	t := Totals{
		SpecOperations:             len(ops),
		UniqueOperationIDs:         len(byID),
		BaselineCommands:           len(Baseline),
		BaselineMapped:             len(MappedBaseline()),
		BaselineExcluded:           len(ExcludedBaseline()),
		BaselineDistinctOperations: len(BaselineTargets()),
	}
	for _, r := range report.Operations {
		switch r.Version {
		case V1:
			t.V1Operations++
		case V2:
			t.V2Operations++
		}
		if r.Deprecated {
			t.DeprecatedOperations++
		}
		if len(r.BaselineCommands) > 0 {
			t.WithBaseline++
		} else {
			t.WithoutBaseline++
		}
		if len(r.Blockers) > 0 {
			t.Blocked++
		}
		if r.Risk.Paginated {
			t.PaginatedOperations++
		}
		if r.Risk.UnboundedCollection {
			t.UnboundedCollections++
		}
		if r.Mutating() {
			t.MutatingOperations++
		}
		if r.Risk.Destructive {
			t.DestructiveOperations++
		}
	}
	t.Unblocked = t.SpecOperations - t.Blocked
	report.Totals = t

	return report, nil
}

// UnknownBaselineTargets returns baseline Ops entries that do not exist in the
// document. A non-empty result means the mapping references an operation the
// current source does not have.
func (r *Report) UnknownBaselineTargets() []string {
	present := map[string]struct{}{}
	for _, rec := range r.Operations {
		present[rec.OperationID] = struct{}{}
	}
	var missing []string
	for _, op := range BaselineTargets() {
		if _, ok := present[op]; !ok {
			missing = append(missing, op)
		}
	}
	sort.Strings(missing)
	return missing
}

// ByFamily groups the records by family, preserving operation order.
func (r *Report) ByFamily() map[Family][]Record {
	out := map[Family][]Record{}
	for _, rec := range r.Operations {
		out[rec.Family] = append(out[rec.Family], rec)
	}
	return out
}

// OperationIDs returns every operationId in the document, sorted.
func (r *Report) OperationIDs() []string {
	out := make([]string, 0, len(r.Operations))
	for _, rec := range r.Operations {
		out = append(out, rec.OperationID)
	}
	sort.Strings(out)
	return out
}
