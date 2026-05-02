package planctx

// PlanShowDiffKeyType is the context key type used to propagate the
// ShowDiff boolean flag through contexts. Using a dedicated package
// avoids import cycles between engine and providers.
type PlanShowDiffKeyType struct{}

// PlanShowDiffKey is the key instance used with context.WithValue to
// store whether a plan should compute/show diffs.
var PlanShowDiffKey PlanShowDiffKeyType
