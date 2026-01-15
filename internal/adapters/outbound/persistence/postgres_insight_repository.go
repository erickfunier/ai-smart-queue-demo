package persistence

import (
	"context"
	"encoding/json"

	"github.com/erickfunier/ai-smart-queue/internal/domain/insights"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresInsightRepository implements insights.InsightRepository using PostgreSQL
type PostgresInsightRepository struct {
	db *pgxpool.Pool
}

// NewPostgresInsightRepository creates a new PostgreSQL insight repository
func NewPostgresInsightRepository(db *pgxpool.Pool) *PostgresInsightRepository {
	return &PostgresInsightRepository{db: db}
}

func (r *PostgresInsightRepository) Create(ctx context.Context, insight *insights.Insight) error {
	// Marshal SuggestedFix to JSON
	suggestedFixJSON, err := json.Marshal(insight.SuggestedFix)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx,
		`INSERT INTO insights (id, job_id, diagnosis, recommendation, suggested_fix, created_at)
         VALUES ($1, $2, $3, $4, $5::jsonb, $6)`,
		insight.ID, insight.JobID, insight.Diagnosis, insight.Recommendation,
		string(suggestedFixJSON), insight.CreatedAt,
	)
	return err
}

func (r *PostgresInsightRepository) GetByID(ctx context.Context, id uuid.UUID) (*insights.Insight, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, job_id, diagnosis, recommendation, suggested_fix, created_at
         FROM insights WHERE id = $1`, id)

	insight := &insights.Insight{}
	var suggestedFixJSON []byte
	err := row.Scan(
		&insight.ID, &insight.JobID, &insight.Diagnosis, &insight.Recommendation,
		&suggestedFixJSON, &insight.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(suggestedFixJSON, &insight.SuggestedFix); err != nil {
		return nil, err
	}

	return insight, nil
}

func (r *PostgresInsightRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) (*insights.Insight, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, job_id, diagnosis, recommendation, suggested_fix, created_at
         FROM insights WHERE job_id = $1 ORDER BY created_at DESC LIMIT 1`, jobID)

	insight := &insights.Insight{}
	var suggestedFixJSON []byte
	err := row.Scan(
		&insight.ID, &insight.JobID, &insight.Diagnosis, &insight.Recommendation,
		&suggestedFixJSON, &insight.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(suggestedFixJSON, &insight.SuggestedFix); err != nil {
		return nil, err
	}

	return insight, nil
}

func (r *PostgresInsightRepository) List(ctx context.Context, limit, offset int) ([]*insights.Insight, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, job_id, diagnosis, recommendation, suggested_fix, created_at
         FROM insights ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var insightsList []*insights.Insight
	for rows.Next() {
		insight := &insights.Insight{}
		var suggestedFixJSON []byte
		err := rows.Scan(
			&insight.ID, &insight.JobID, &insight.Diagnosis, &insight.Recommendation,
			&suggestedFixJSON, &insight.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(suggestedFixJSON, &insight.SuggestedFix); err != nil {
			return nil, err
		}

		insightsList = append(insightsList, insight)
	}

	return insightsList, nil
}

func (r *PostgresInsightRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM insights WHERE id = $1`, id)
	return err
}
