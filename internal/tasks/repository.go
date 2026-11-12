package tasks

import (
	"context"
	"database/sql"
)

type Repository interface {
	List(ctx context.Context, userID int64, opts listOptions) ([]Task, int64, error)
	Get(ctx context.Context, userID, taskID int64) (Task, error)
	Create(ctx context.Context, userID int64, title, description string) (Task, error)
	Update(ctx context.Context, userID, taskID int64, title, description, status string) (Task, error)
	Delete(ctx context.Context, userID, taskID int64) (bool, error)
}

type PostgresRepository struct {
	DB *sql.DB
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(row rowScanner) (Task, error) {
	var task Task
	err := row.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	return task, err
}

func (p *PostgresRepository) List(ctx context.Context, userID int64, opts listOptions) ([]Task, int64, error) {
	var total int64
	err := p.DB.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM tasks
         WHERE user_id = $1 AND ($2::text = '' OR status = $2::text)`,
		userID, opts.Status,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	orderBy := "created_at DESC, id DESC"
	if opts.Sort == "oldest" {
		orderBy = "created_at ASC, id ASC"
	}

	query := `SELECT id, title, description, status, created_at, updated_at
        FROM tasks
        WHERE user_id = $1 AND ($2::text = '' OR status = $2::text)
        ORDER BY ` + orderBy + ` LIMIT $3 OFFSET $4`

	offset := (opts.Page - 1) * opts.Limit
	rows, err := p.DB.QueryContext(ctx, query, userID, opts.Status, opts.Limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (p *PostgresRepository) Get(ctx context.Context, userID, taskID int64) (Task, error) {
	return scanTask(p.DB.QueryRowContext(
		ctx,
		`SELECT id, title, description, status, created_at, updated_at
         FROM tasks WHERE id = $1 AND user_id = $2`,
		taskID, userID,
	))
}

func (p *PostgresRepository) Create(ctx context.Context, userID int64, title, description string) (Task, error) {
	return scanTask(p.DB.QueryRowContext(
		ctx,
		`INSERT INTO tasks (user_id, title, description)
         VALUES ($1, $2, $3)
         RETURNING id, title, description, status, created_at, updated_at`,
		userID, title, description,
	))
}

func (p *PostgresRepository) Update(ctx context.Context, userID, taskID int64, title, description, status string) (Task, error) {
	return scanTask(p.DB.QueryRowContext(
		ctx,
		`UPDATE tasks
         SET title = $1, description = $2, status = $3, updated_at = NOW()
         WHERE id = $4 AND user_id = $5
         RETURNING id, title, description, status, created_at, updated_at`,
		title, description, status, taskID, userID,
	))
}

func (p *PostgresRepository) Delete(ctx context.Context, userID, taskID int64) (bool, error) {
	result, err := p.DB.ExecContext(
		ctx,
		`DELETE FROM tasks WHERE id = $1 AND user_id = $2`,
		taskID, userID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}
