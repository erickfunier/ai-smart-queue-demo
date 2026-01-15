package persistence

import (
	"context"

	"github.com/erickfunier/ai-smart-queue/internal/domain/queue"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresJobRepository implements queue.JobRepository using PostgreSQL
type PostgresJobRepository struct {
	db *pgxpool.Pool
}

// NewPostgresJobRepository creates a new PostgreSQL job repository
func NewPostgresJobRepository(db *pgxpool.Pool) *PostgresJobRepository {
	return &PostgresJobRepository{db: db}
}

func (r *PostgresJobRepository) Create(ctx context.Context, job *queue.Job) error {
	var payload interface{}
	if job.Payload != nil {
		// Convert []byte to string for JSONB column
		payload = string(job.Payload)
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO jobs (id, queue, type, status, attempts, payload, scheduled_for, created_at, updated_at, error)
         VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10)`,
		job.ID, job.Queue, job.Type, job.Status, job.Attempts,
		payload, job.ScheduledFor, job.CreatedAt, job.UpdatedAt, job.Error,
	)
	return err
}

func (r *PostgresJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*queue.Job, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, queue, type, status, attempts, payload, scheduled_for, created_at, updated_at, error
         FROM jobs WHERE id = $1`, id)

	job := &queue.Job{}
	err := row.Scan(
		&job.ID, &job.Queue, &job.Type, &job.Status, &job.Attempts,
		&job.Payload, &job.ScheduledFor, &job.CreatedAt, &job.UpdatedAt, &job.Error,
	)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (r *PostgresJobRepository) Update(ctx context.Context, job *queue.Job) error {
	var payload interface{}
	if job.Payload != nil {
		// Convert []byte to string for JSONB column
		payload = string(job.Payload)
	}

	_, err := r.db.Exec(ctx,
		`UPDATE jobs SET status=$1, attempts=$2, payload=$3::jsonb, scheduled_for=$4, updated_at=$5, error=$6
         WHERE id=$7`,
		job.Status, job.Attempts, payload, job.ScheduledFor, job.UpdatedAt, job.Error, job.ID,
	)
	return err
}

func (r *PostgresJobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, id)
	return err
}

func (r *PostgresJobRepository) FindPendingJobs(ctx context.Context, queueName string, limit int) ([]*queue.Job, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, queue, type, status, attempts, payload, scheduled_for, created_at, updated_at, error
         FROM jobs 
         WHERE queue = $1 AND status IN ($2, $3)
         AND (scheduled_for IS NULL OR scheduled_for <= NOW())
         ORDER BY created_at ASC
         LIMIT $4`,
		queueName, queue.StatusPending, queue.StatusRetrying, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*queue.Job
	for rows.Next() {
		job := &queue.Job{}
		err := rows.Scan(
			&job.ID, &job.Queue, &job.Type, &job.Status, &job.Attempts,
			&job.Payload, &job.ScheduledFor, &job.CreatedAt, &job.UpdatedAt, &job.Error,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (r *PostgresJobRepository) FindByStatus(ctx context.Context, status queue.Status, limit int) ([]*queue.Job, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, queue, type, status, attempts, payload, scheduled_for, created_at, updated_at, error
         FROM jobs WHERE status = $1 LIMIT $2`,
		status, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*queue.Job
	for rows.Next() {
		job := &queue.Job{}
		err := rows.Scan(
			&job.ID, &job.Queue, &job.Type, &job.Status, &job.Attempts,
			&job.Payload, &job.ScheduledFor, &job.CreatedAt, &job.UpdatedAt, &job.Error,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (r *PostgresJobRepository) CountByStatus(ctx context.Context, status queue.Status) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM jobs WHERE status = $1`, status,
	).Scan(&count)
	return count, err
}

func (r *PostgresJobRepository) GetDLQJobs(ctx context.Context, limit, offset int) ([]*queue.Job, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, queue, type, status, attempts, payload, scheduled_for, created_at, updated_at, error
         FROM jobs 
         WHERE status = $1 AND attempts >= 3
         ORDER BY updated_at DESC
         LIMIT $2 OFFSET $3`,
		queue.StatusFailed, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*queue.Job
	for rows.Next() {
		job := &queue.Job{}
		err := rows.Scan(
			&job.ID, &job.Queue, &job.Type, &job.Status, &job.Attempts,
			&job.Payload, &job.ScheduledFor, &job.CreatedAt, &job.UpdatedAt, &job.Error,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (r *PostgresJobRepository) MoveToDLQ(ctx context.Context, jobID uuid.UUID) error {
	// In this implementation, we keep failed jobs in the same table
	// but could move to a separate dlq table if needed
	_, err := r.db.Exec(ctx,
		`UPDATE jobs SET status = $1, updated_at = NOW() WHERE id = $2`,
		queue.StatusFailed, jobID,
	)
	return err
}

func (r *PostgresJobRepository) CountDLQJobs(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM jobs WHERE status = $1 AND attempts >= 3`,
		queue.StatusFailed,
	).Scan(&count)
	return count, err
}
