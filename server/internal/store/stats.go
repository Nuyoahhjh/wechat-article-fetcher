package store

import (
	"context"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

func (s *Store) DashboardCounts(ctx context.Context, dayStart time.Time) (DashboardData, error) {
	var counts DashboardData
	var succeeded, failed int
	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM agents),
			(SELECT COUNT(*) FROM agents WHERE status IN ('online', 'busy')),
			(SELECT COUNT(*) FROM tasks),
			(SELECT COUNT(*) FROM tasks WHERE status IN ('pending', 'assigned', 'running')),
			(SELECT COUNT(*) FROM articles),
			(SELECT COUNT(*) FROM articles WHERE created_at >= ?),
			(SELECT COUNT(*) FROM tasks WHERE status = 'succeeded'),
			(SELECT COUNT(*) FROM tasks WHERE status = 'failed')`,
		dayStart.UTC(),
	).Scan(
		&counts.Users,
		&counts.Agents,
		&counts.OnlineAgents,
		&counts.Tasks,
		&counts.ActiveTasks,
		&counts.Articles,
		&counts.TodayArticles,
		&succeeded,
		&failed,
	)
	if err != nil {
		return DashboardData{}, err
	}
	counts.SucceededTasks = succeeded
	counts.FailedTasks = failed
	if succeeded+failed > 0 {
		counts.TaskSuccessRate = float64(succeeded) * 100 / float64(succeeded+failed)
	}
	counts.Database = "ok"
	trendStart := dayStart.AddDate(0, 0, -13)
	tasksByDay, err := s.countsByDay(
		ctx,
		"tasks",
		trendStart,
	)
	if err != nil {
		return DashboardData{}, err
	}
	articlesByDay, err := s.countsByDay(
		ctx,
		"articles",
		trendStart,
	)
	if err != nil {
		return DashboardData{}, err
	}
	counts.Trends = make([]domain.DailyCount, 0, 14)
	for offset := 0; offset < 14; offset++ {
		day := trendStart.AddDate(0, 0, offset).Format("2006-01-02")
		counts.Trends = append(counts.Trends, domain.DailyCount{
			Date:     day,
			Tasks:    tasksByDay[day],
			Articles: articlesByDay[day],
		})
	}
	return counts, nil
}

func (s *Store) countsByDay(
	ctx context.Context,
	table string,
	start time.Time,
) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT substr(created_at, 1, 10), COUNT(*)
		FROM `+table+`
		WHERE created_at >= ?
		GROUP BY substr(created_at, 1, 10)`, start.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		result[day] = count
	}
	return result, rows.Err()
}

type DashboardData struct {
	Users           int                 `json:"users"`
	Agents          int                 `json:"agents"`
	OnlineAgents    int                 `json:"online_agents"`
	Tasks           int                 `json:"tasks"`
	ActiveTasks     int                 `json:"active_tasks"`
	Articles        int                 `json:"articles"`
	TodayArticles   int                 `json:"today_articles"`
	TaskSuccessRate float64             `json:"task_success_rate"`
	SucceededTasks  int                 `json:"succeeded_tasks"`
	FailedTasks     int                 `json:"failed_tasks"`
	Trends          []domain.DailyCount `json:"trends"`
	Database        string              `json:"database"`
}

func (s *Store) CountAgents(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agents`).Scan(&count)
	return count, err
}

func (s *Store) CountTasks(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&count)
	return count, err
}
