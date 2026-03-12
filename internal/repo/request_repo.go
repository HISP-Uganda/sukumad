package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sukumad/internal/types"
)

type RequestRepo struct {
	db *pgxpool.Pool
}

func NewRequestRepo(db *pgxpool.Pool) *RequestRepo { return &RequestRepo{db: db} }

func (r *RequestRepo) List(ctx context.Context, limit, offset int) ([]types.Request, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, uid, source, destination, cc_servers, depends_on, batchid,
		       url_suffix, body, body_is_query_param,
		       frequency_type, period, week, month, year,
		       msisdn, raw_msg, facility, district, report_type, object_type, extras,
		       suspended, idempotency_key, created, updated
		  FROM requests
		  ORDER BY id DESC
		  LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []types.Request
	for rows.Next() {
		var v types.Request
		if err := rows.Scan(
			&v.ID, &v.UID, &v.Source, &v.Destination, &v.CCServers, &v.DependsOn, &v.BatchID,
			&v.URLSuffix, &v.Body, &v.BodyIsQueryParam,
			&v.FrequencyType, &v.Period, &v.Week, &v.Month, &v.Year,
			&v.MSISDN, &v.RawMsg, &v.Facility, &v.District, &v.ReportType, &v.ObjectType, &v.Extras,
			&v.Suspended, &v.IdempotencyKey, &v.Created, &v.Updated,
		); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RequestRepo) Get(ctx context.Context, id int64) (*types.Request, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, uid, source, destination, cc_servers, depends_on, batchid,
		       url_suffix, body, body_is_query_param,
		       frequency_type, period, week, month, year,
		       msisdn, raw_msg, facility, district, report_type, object_type, extras,
		       suspended, idempotency_key, created, updated
		  FROM requests WHERE id=$1`, id)
	var v types.Request
	if err := row.Scan(
		&v.ID, &v.UID, &v.Source, &v.Destination, &v.CCServers, &v.DependsOn, &v.BatchID,
		&v.URLSuffix, &v.Body, &v.BodyIsQueryParam,
		&v.FrequencyType, &v.Period, &v.Week, &v.Month, &v.Year,
		&v.MSISDN, &v.RawMsg, &v.Facility, &v.District, &v.ReportType, &v.ObjectType, &v.Extras,
		&v.Suspended, &v.IdempotencyKey, &v.Created, &v.Updated,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	return &v, nil
}

func (r *RequestRepo) Create(ctx context.Context, v *types.Request) (*types.Request, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO requests (source, destination, cc_servers, depends_on, batchid,
			url_suffix, body, body_is_query_param, frequency_type, period, week, month, year,
			msisdn, raw_msg, facility, district, report_type, object_type, extras,
			suspended, idempotency_key)
		VALUES ($1,$2,$3,$4,$5,
		        $6,$7,$8,$9,$10,$11,$12,$13,
		        $14,$15,$16,$17,$18,$19,$20,
		        $21,$22)
		RETURNING id, uid, created, updated`,
		v.Source, v.Destination, v.CCServers, v.DependsOn, v.BatchID,
		v.URLSuffix, v.Body, v.BodyIsQueryParam, v.FrequencyType, v.Period, v.Week, v.Month, v.Year,
		v.MSISDN, v.RawMsg, v.Facility, v.District, v.ReportType, v.ObjectType, v.Extras,
		v.Suspended, v.IdempotencyKey,
	)
	if err := row.Scan(&v.ID, &v.UID, &v.Created, &v.Updated); err != nil {
		return nil, err
	}
	return v, nil
}

func (r *RequestRepo) Update(ctx context.Context, v *types.Request) (*types.Request, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE requests SET
			source=$1, destination=$2, cc_servers=$3, depends_on=$4, batchid=$5,
			url_suffix=$6, body=$7, body_is_query_param=$8, frequency_type=$9, period=$10, week=$11, month=$12, year=$13,
			msisdn=$14, raw_msg=$15, facility=$16, district=$17, report_type=$18, object_type=$19, extras=$20,
			suspended=$21, idempotency_key=$22, updated=now()
		WHERE id=$23
		RETURNING updated`,
		v.Source, v.Destination, v.CCServers, v.DependsOn, v.BatchID,
		v.URLSuffix, v.Body, v.BodyIsQueryParam, v.FrequencyType, v.Period, v.Week, v.Month, v.Year,
		v.MSISDN, v.RawMsg, v.Facility, v.District, v.ReportType, v.ObjectType, v.Extras,
		v.Suspended, v.IdempotencyKey, v.ID,
	)
	if err := row.Scan(&v.Updated); err != nil {
		return nil, err
	}
	return v, nil
}

func (r *RequestRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM requests WHERE id=$1`, id)
	return err
}

func (r *RequestRepo) Expand(ctx context.Context, requestID int64, dest int32, ccIDs []int) error {
	// convert []int → integer[]
	var cc any
	if len(ccIDs) > 0 {
		cc = ccIDs
	} else {
		cc = nil
	}
	_, err := r.db.Exec(ctx, `SELECT expand_request_to_deliveries($1,$2,$3,$4,$5)`, requestID, dest, cc, 100, 50)
	return err
}
