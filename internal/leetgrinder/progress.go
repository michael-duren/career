package leetgrinder

type Progress struct {
	Solved      int
	Independent int
	Completed   int
	NextDay     int
}

func Summarize(state State) Progress {
	solved, independent := map[string]bool{}, map[string]bool{}
	for _, a := range state.Attempts {
		if a.Outcome == "solved" {
			solved[a.ProblemSlug] = true
			if !a.Assisted {
				independent[a.ProblemSlug] = true
			}
		}
	}
	completed := map[int]bool{}
	for _, n := range state.CompletedDays {
		completed[n] = true
	}
	next := 0
	for n := 1; n <= 84; n++ {
		if !completed[n] {
			next = n
			break
		}
	}
	return Progress{Solved: len(solved), Independent: len(independent), Completed: len(completed), NextDay: next}
}

func (s State) DayCompleted(n int) bool {
	for _, day := range s.CompletedDays {
		if day == n {
			return true
		}
	}
	return false
}
func (s State) ProblemAttempts(slug string) []Attempt {
	result := []Attempt{}
	for _, a := range s.Attempts {
		if a.ProblemSlug == slug {
			result = append(result, a)
		}
	}
	return result
}
func (s State) ProblemStatus(slug string) string {
	attempts := s.ProblemAttempts(slug)
	assisted := false
	for _, a := range attempts {
		if a.Outcome == "solved" {
			if !a.Assisted {
				return "Solved independently"
			}
			assisted = true
		}
	}
	if assisted {
		return "Solved with help"
	}
	if len(attempts) > 0 {
		if attempts[0].Outcome == "struggled" {
			return "Struggled"
		}
		return "Unfinished"
	}
	return "Not attempted"
}
