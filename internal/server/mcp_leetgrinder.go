package server

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type leetgrinderReviewsInput struct {
	Days  int `json:"days,omitempty" jsonschema:"also list reviews due within this many days after today, 0-30 (default 7)"`
	Limit int `json:"limit,omitempty" jsonschema:"maximum problems, 1-100 (default 30)"`
}

type leetgrinderStatsInput struct {
	Weeks int `json:"weeks,omitempty" jsonschema:"weeks of trends, 1-26 (default 12)"`
}

type logLeetgrinderAttemptInput struct {
	ID              string `json:"id,omitempty" jsonschema:"optional attempt UUID; pass the same ID to retry without logging twice"`
	Problem         string `json:"problem" jsonschema:"LeetCode or NeetCode problem link, or LeetCode slug"`
	Outcome         string `json:"outcome" jsonschema:"solved, struggled, or unfinished"`
	Minutes         int    `json:"minutes" jsonschema:"minutes spent, 1-240"`
	Assisted        bool   `json:"assisted,omitempty" jsonschema:"true when hints or a solution were used"`
	Notes           string `json:"notes,omitempty" jsonschema:"what to remember next time, up to 2000 characters"`
	TimeComplexity  string `json:"timeComplexity,omitempty" jsonschema:"stated time complexity such as O(n log n); required for solved and struggled"`
	SpaceComplexity string `json:"spaceComplexity,omitempty" jsonschema:"stated space complexity such as O(n); required for solved and struggled"`
	Code            string `json:"code,omitempty" jsonschema:"optional solution code, up to 64 KiB"`
	CodeLanguage    string `json:"codeLanguage,omitempty" jsonschema:"LeetCode language slug of code, such as python3 or golang; required with code"`
	WantsReview     bool   `json:"wantsReview,omitempty" jsonschema:"true to review the problem again soon"`
	Approach        string `json:"approach,omitempty" jsonschema:"optimal, suboptimal (a simpler approach for time), or empty"`
}

// mcpLeetgrinderToday is the learner's day, planned like the dashboard.
func (s *Server) mcpLeetgrinderToday(ctx context.Context) (leetgrinder.Today, error) {
	today, err := s.db.LeetgrinderToday(ctx, s.clock())
	if err != nil {
		return today, toolError(err)
	}
	return today, nil
}

func mcpLeetgrinderProblem(p leetgrinder.Problem) map[string]any {
	return map[string]any{"slug": p.Slug, "title": p.DisplayTitle(), "number": p.Number, "difficulty": p.Difficulty, "topics": p.Topics, "url": p.URL()}
}

// mcpLeetgrinderCard describes a review card as of now.
func mcpLeetgrinderCard(today leetgrinder.Today, c leetgrinder.Card) map[string]any {
	loc := today.Settings.Location()
	out := mcpLeetgrinderProblem(c.Problem)
	out["dueDate"] = leetgrinder.Date(c.Due, loc).Format(time.DateOnly)
	out["due"] = today.Due(c)
	out["recall"] = c.Retrievability(today.Now)
	out["reason"] = leetgrinder.ReviewReason(c, today.Now, loc)
	out["flagged"] = c.Flagged()
	out["lastOutcome"] = c.Last.Outcome
	out["lastAttemptedAt"] = c.Last.CreatedAt
	return out
}

func (s *Server) addLeetgrinderTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_leetgrinder_today", Description: "Today's Leetgrinder practice: the daily goal and progress (new, review, bonus), whether it is met, the streak, today's review picks with their reasons and recall, today's new problems picked from todos (when that option is on), and how many more reviews are due. Like opening the dashboard, the first access of a day freezes that day's goal and picks.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		today, err := s.mcpLeetgrinderToday(ctx)
		if err != nil {
			return nil, nil, err
		}
		target := today.Progress.Target()
		picks := make([]map[string]any, 0, len(today.Reviews))
		for _, r := range today.Reviews {
			pick := mcpLeetgrinderProblem(r.Problem)
			pick["reason"], pick["done"] = r.Reason, r.Done
			pick["recall"] = r.Card.Retrievability(today.Now)
			picks = append(picks, pick)
		}
		newPicks := make([]map[string]any, 0, len(today.NewPicks))
		for _, p := range today.NewPicks {
			pick := mcpLeetgrinderProblem(p.Problem)
			pick["set"], pick["done"] = p.SetTitle, p.Done
			newPicks = append(newPicks, pick)
		}
		return nil, map[string]any{
			"date":           today.Date.Format(time.DateOnly),
			"timezone":       today.Settings.Timezone,
			"goal":           map[string]int{"new": target.New, "review": target.Review},
			"done":           map[string]int{"new": today.Progress.New(), "review": today.Progress.Reviews(), "bonus": today.Progress.Bonus(), "practice": today.Progress.Practice()},
			"met":            today.Progress.Met(),
			"remaining":      today.Progress.Remaining(),
			"streak":         map[string]int{"current": today.Streaks.Current, "longest": today.Streaks.Longest},
			"reviewPicks":    picks,
			"newPicks":       newPicks,
			"newPicksFailed": today.NewPicksFailed,
			"alsoDue":        today.Backlog(),
		}, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_leetgrinder_reviews", Description: "Upcoming Leetgrinder reviews: problems due by the end of today (today's picks first, then flagged, lowest recall and most overdue) followed by those due within the next days, soonest first, each with its due date, estimated recall, flag reason and last outcome.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in leetgrinderReviewsInput) (*mcp.CallToolResult, any, error) {
		days, limit := 7, 30
		if in.Days != 0 {
			days = in.Days
		}
		if in.Limit != 0 {
			limit = in.Limit
		}
		if days < 0 || days > 30 || limit < 1 || limit > 100 {
			return nil, nil, invalidInput("days must be 0-30 and limit 1-100")
		}
		today, err := s.mcpLeetgrinderToday(ctx)
		if err != nil {
			return nil, nil, err
		}
		cards := leetgrinder.UpcomingReviews(today, days)
		out := make([]map[string]any, 0, min(limit, len(cards)))
		for _, c := range cards[:min(limit, len(cards))] {
			card := mcpLeetgrinderCard(today, c)
			card["todaysPick"] = today.Picked(c.Problem.Slug)
			card["attemptedToday"] = today.AttemptedOn(c.Problem.Slug)
			out = append(out, card)
		}
		return nil, map[string]any{"date": today.Date.Format(time.DateOnly), "days": days, "reviews": out, "total": len(cards)}, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_leetgrinder_stats", Description: "Leetgrinder strengths and weaknesses: per LeetCode topic (weakest recall first, topics with fewer than 3 problems grouped under Other) the problems attempted and solved, struggle rate, average recall and due count; attempted and solved per difficulty; and weekly trends (problems solved, independent-solve rate, average minutes by difficulty, complexity-check accuracy). Use it to find weak topics, for example before building a todo set with create_leetgrinder_todo_set.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in leetgrinderStatsInput) (*mcp.CallToolResult, any, error) {
		weeks := 12
		if in.Weeks != 0 {
			weeks = in.Weeks
		}
		if weeks < 1 || weeks > 26 {
			return nil, nil, invalidInput("weeks must be 1-26")
		}
		today, err := s.mcpLeetgrinderToday(ctx)
		if err != nil {
			return nil, nil, err
		}
		page := leetgrinder.NewStatsPage(today, leetgrinder.ParseStatsFilter(nil), weeks)
		topics := make([]map[string]any, 0, len(page.Topics))
		for _, t := range page.Topics {
			topics = append(topics, map[string]any{"topic": t.Key, "label": t.Label, "problems": t.Problems, "solved": t.Solved, "attempts": t.Attempts, "struggleRate": t.StruggleRate(), "recall": t.Recall(), "due": t.Due})
		}
		difficulties := make([]map[string]any, 0, len(page.Difficulties))
		for _, d := range page.Difficulties {
			difficulties = append(difficulties, map[string]any{"difficulty": leetgrinder.DifficultyLabel(d.Difficulty), "attempted": d.Attempted, "solved": d.Solved})
		}
		trends := make([]map[string]any, 0, len(page.Trends.Weeks))
		for _, w := range page.Trends.Weeks {
			week := map[string]any{"weekOf": w.Start.Format(time.DateOnly), "problemsSolved": w.Solved, "solves": w.Solves, "complexityChecks": w.Checked}
			if rate, ok := w.IndependentRate(); ok {
				week["independentRate"] = rate
			}
			if acc, ok := w.Accuracy(); ok {
				week["complexityAccuracy"] = acc
			}
			minutes := map[string]int{}
			for _, d := range leetgrinder.TrendDifficulties {
				if m, ok := w.AverageMinutes(d); ok {
					minutes[d] = m
				}
			}
			week["averageMinutes"] = minutes
			trends = append(trends, week)
		}
		return nil, map[string]any{"topics": topics, "difficulties": difficulties, "weeklyTrends": trends}, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "log_leetgrinder_attempt", Description: "Log a Leetgrinder attempt from chat, for a problem solved on a whiteboard, in an IDE or a mock interview. It counts as new, review or practice like any attempt, dated now. Time and space complexity are required for solved and struggled attempts. Pass the same id to retry without logging twice. Only log when the user asked to. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Log Leetgrinder attempt", DestructiveHint: new(bool), IdempotentHint: true, OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in logLeetgrinderAttemptInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		slug, err := leetgrinder.ResolveProblemRef(in.Problem)
		if err != nil {
			return nil, nil, refInputError(in.Problem, err)
		}
		if in.ID == "" {
			in.ID = uuid.NewString()
		}
		attempt, status, message := newLeetgrinderAttempt(leetgrinderAPIAttemptInput{
			ID: in.ID, ProblemSlug: slug, Outcome: in.Outcome, Minutes: in.Minutes, Assisted: in.Assisted, Notes: in.Notes,
			TimeComplexity: in.TimeComplexity, SpaceComplexity: in.SpaceComplexity, Code: in.Code, CodeLanguage: in.CodeLanguage, WantsReview: in.WantsReview, Approach: in.Approach,
		}, "mcp")
		if status != 0 {
			return nil, nil, invalidInput("%s", message)
		}
		// As for the extension: plan today first, so a first access that is
		// itself a review still plans that review.
		_ = s.db.PlanLeetgrinderToday(ctx, s.clock())
		saved, err := s.db.SaveLeetgrinderAttemptWithProblem(ctx, attempt, "", nil, s.clock())
		switch {
		case errors.Is(err, database.ErrConflict):
			return nil, nil, invalidInput("an attempt with this id was saved earlier with different values; omit id to log a new attempt")
		case errors.Is(err, database.ErrInvalid):
			return nil, nil, invalidInput("check the attempt fields and try again")
		case err != nil:
			return nil, nil, toolError(err)
		}
		kind := ""
		if today, err := s.db.LeetgrinderToday(ctx, s.clock()); err == nil {
			kind = today.Kind(saved.ProblemSlug)
		}
		return nil, map[string]any{"id": saved.ID, "problemSlug": saved.ProblemSlug, "outcome": saved.Outcome, "minutes": saved.Minutes, "kind": kind, "historyUrl": leetgrinder.ProblemURL(saved.ProblemSlug)}, nil
	})
}
