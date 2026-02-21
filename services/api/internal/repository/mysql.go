package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/dysania/meowlator/services/api/internal/domain"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(dsn string) (*MySQLRepository, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &MySQLRepository{db: db}, nil
}

func (r *MySQLRepository) Close() error {
	if r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *MySQLRepository) CreateSample(ctx context.Context, sample *domain.Sample) error {
	if sample.CatID == "" {
		sample.CatID = "cat-default"
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO users (user_id, wechat_openid) VALUES (?, NULL)
		 ON DUPLICATE KEY UPDATE user_id = user_id`, sample.UserID,
	)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO cats (cat_id, user_id, name) VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE user_id = VALUES(user_id)`,
		sample.CatID, sample.UserID, "猫猫",
	)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO samples (sample_id, user_id, cat_id, image_key, scene_tag, model_version, created_at, expire_at)
		 VALUES (?, ?, ?, ?, ?, ?, FROM_UNIXTIME(?), FROM_UNIXTIME(?))`,
		sample.SampleID, sample.UserID, sample.CatID, sample.ImageKey, sample.SceneTag, sample.ModelVersion, sample.CreatedAt, sample.ExpireAt,
	)
	if err != nil {
		if isDuplicate(err) {
			return domain.ErrConflict
		}
		return err
	}
	return tx.Commit()
}

func (r *MySQLRepository) GetSample(ctx context.Context, sampleID string) (*domain.Sample, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT sample_id, user_id, cat_id, image_key, scene_tag, model_version,
			edge_pred_json, final_pred_json,
			UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(expire_at)
		 FROM samples WHERE sample_id = ?`, sampleID)

	var sample domain.Sample
	var sceneTag sql.NullString
	var edgePredJSON, finalPredJSON sql.NullString
	if err := row.Scan(
		&sample.SampleID,
		&sample.UserID,
		&sample.CatID,
		&sample.ImageKey,
		&sceneTag,
		&sample.ModelVersion,
		&edgePredJSON,
		&finalPredJSON,
		&sample.CreatedAt,
		&sample.ExpireAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if sceneTag.Valid {
		sample.SceneTag = sceneTag.String
	}
	if edgePredJSON.Valid && edgePredJSON.String != "" {
		var pred domain.InferenceResult
		if err := json.Unmarshal([]byte(edgePredJSON.String), &pred); err == nil {
			sample.EdgePred = &pred
		}
	}
	if finalPredJSON.Valid && finalPredJSON.String != "" {
		var pred domain.InferenceResult
		if err := json.Unmarshal([]byte(finalPredJSON.String), &pred); err == nil {
			sample.FinalPred = &pred
		}
	}
	return &sample, nil
}

func (r *MySQLRepository) UpdatePredictions(ctx context.Context, sampleID string, edgePred, finalPred *domain.InferenceResult, sceneTag string, modelVersion string) error {
	var edgeJSON any
	var finalJSON any
	if edgePred != nil {
		raw, err := json.Marshal(edgePred)
		if err != nil {
			return err
		}
		edgeJSON = string(raw)
	}
	if finalPred != nil {
		raw, err := json.Marshal(finalPred)
		if err != nil {
			return err
		}
		finalJSON = string(raw)
	}

	result, err := r.db.ExecContext(ctx,
		`UPDATE samples
		 SET edge_pred_json = COALESCE(?, edge_pred_json),
		     final_pred_json = COALESCE(?, final_pred_json),
		     scene_tag = CASE WHEN ? = '' THEN scene_tag ELSE ? END,
		     model_version = CASE WHEN ? = '' THEN model_version ELSE ? END
		 WHERE sample_id = ?`,
		edgeJSON, finalJSON,
		sceneTag, sceneTag,
		modelVersion, modelVersion,
		sampleID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MySQLRepository) SaveFeedback(ctx context.Context, fb *domain.Feedback) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO feedback (feedback_id, sample_id, user_id, is_correct, true_label, reliability_score, training_weight, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, FROM_UNIXTIME(?))`,
		fb.FeedbackID,
		fb.SampleID,
		fb.UserID,
		fb.IsCorrect,
		string(fb.TrueLabel),
		fb.ReliabilityScore,
		fb.TrainingWeight,
		fb.CreatedAt,
	)
	if err != nil {
		if isDuplicate(err) {
			return domain.ErrConflict
		}
		if stringsContains(err.Error(), "foreign key") {
			return domain.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *MySQLRepository) DeleteSample(ctx context.Context, sampleID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `DELETE FROM feedback WHERE sample_id = ?`, sampleID)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM samples WHERE sample_id = ?`, sampleID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit()
}

func (r *MySQLRepository) UserFeedbackStats(ctx context.Context, userID string) (total int, extremeRatio float64, suspicious bool) {
	row := r.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*) AS total,
			SUM(CASE WHEN true_label IN ('DEFENSIVE_ALERT', 'UNCERTAIN') THEN 1 ELSE 0 END) AS extreme_count,
			SUM(CASE WHEN is_correct = false THEN 1 ELSE 0 END) AS conflict_count
		 FROM feedback WHERE user_id = ?`,
		userID,
	)
	var extremeCount, conflictCount sql.NullInt64
	if err := row.Scan(&total, &extremeCount, &conflictCount); err != nil {
		return 0, 0, false
	}
	if total == 0 {
		return 0, 0, false
	}
	extreme := 0
	conflicts := 0
	if extremeCount.Valid {
		extreme = int(extremeCount.Int64)
	}
	if conflictCount.Valid {
		conflicts = int(conflictCount.Int64)
	}
	extremeRatio = float64(extreme) / float64(total)
	suspicious = total >= 3 && conflicts == total
	return total, extremeRatio, suspicious
}

func isDuplicate(err error) bool {
	return stringsContains(err.Error(), "duplicate entry")
}

func stringsContains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
