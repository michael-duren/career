package leetgrinder

// OptimalForm is the "Edit optimal complexity" form. Revision is the
// Problem.OptimalRevision the form was rendered from.
type OptimalForm struct {
	Time, Space ComplexityInput
	Note        string
	Revision    string
	Error       string
}

// NewOptimalForm prefills the form with the problem's current optimum.
func NewOptimalForm(p Problem) OptimalForm {
	return OptimalForm{Time: NewComplexityInput(p.OptimalTime), Space: NewComplexityInput(p.OptimalSpace), Note: p.OptimalNote, Revision: p.OptimalRevision()}
}

// OptimalEditURL is the page that edits slug's optimal complexity; it also
// receives the form post.
func OptimalEditURL(slug string) string { return ProblemURL(slug) + "/optimal" }

// OptimalReestimateURL receives the "Re-estimate" post.
func OptimalReestimateURL(slug string) string { return OptimalEditURL(slug) + "/reestimate" }
