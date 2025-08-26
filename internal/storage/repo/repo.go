package repo

import (
	"context"
	"database/sql"
)

type Repo struct {
	Driver *sql.DB
}

func New(driver *sql.DB) *Repo {
	return &Repo{driver}
}

func (r *Repo) Close() error {
	return r.Driver.Close()
}

type DailyDataRow struct {
	Date   string
	Level  float64
	Volume float64
}

// GetDailyDataCount fetches the count of unique days for a given reservoir and date range.
func (r *Repo) GetDailyDataCount(ctx context.Context, reservoirID int, dateFrom, dateTo string) (int64, error) {
	query := `
        SELECT COUNT(DISTINCT DATE(` + "`date`" + `)) 
        FROM daily_values 
        WHERE reservoir_id = ? 
          AND category = 'level' 
          AND ` + "`date`" + ` BETWEEN ? AND ?
    `
	var count int64
	err := r.Driver.QueryRowContext(ctx, query, reservoirID, dateFrom, dateTo).Scan(&count)
	return count, err
}

// GetDailyData fetches paginated, pivoted data for a given reservoir and date range.
func (r *Repo) GetDailyData(ctx context.Context, reservoirID int, dateFrom, dateTo string, page, perPage int) ([]DailyDataRow, error) {
	// NOTE: The original PHP code hardcodes `reservoir_id = 1` in this query.
	// We are replicating that behavior here. If this is a bug, change `d.reservoir_id = 1`
	// to `d.reservoir_id = ?` and add `reservoirID` to the query arguments.
	query := `
        SELECT 
            DATE(d.` + "`date`" + `) as date,
            MAX(CASE WHEN d.category = 'level' THEN d.value END) as level,
            MAX(CASE WHEN d.category = 'volume' THEN d.value END) as volume
        FROM daily_values d
        WHERE d.reservoir_id = ? 
          AND d.` + "`date`" + ` BETWEEN ? AND ?
        GROUP BY DATE(d.` + "`date`" + `)
        ORDER BY DATE(d.` + "`date`" + `)
        LIMIT ? OFFSET ?
    `
	offset := (page - 1) * perPage

	rows, err := r.Driver.QueryContext(ctx, query, reservoirID, dateFrom, dateTo, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DailyDataRow
	for rows.Next() {
		var row DailyDataRow
		// Use sql.NullFloat64 to handle cases where a value might be missing for a day
		var level, volume sql.NullFloat64
		if err := rows.Scan(&row.Date, &level, &volume); err != nil {
			return nil, err
		}
		row.Level = level.Float64   // Defaults to 0 if NULL
		row.Volume = volume.Float64 // Defaults to 0 if NULL
		results = append(results, row)
	}

	return results, rows.Err()
}
