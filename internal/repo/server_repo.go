package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sukumad/internal/types"
)

type ServerRepo struct {
	db *pgxpool.Pool
}

func NewServerRepo(db *pgxpool.Pool) *ServerRepo { return &ServerRepo{db: db} }

func (r *ServerRepo) List(ctx context.Context, limit, offset int) ([]types.Server, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, uid, name, username, password, auth_token, ipaddress, url, callback_url, cc_urls,
		       http_method, auth_method, system_type, endpoint_type, url_params, allow_callbacks,
		       allow_copies, use_async, use_ssl, parse_responses, ssl_client_certkey_file,
		       start_submission_period, end_submission_period, xml_response_xpath, json_response_xpath, suspended,
		       rps, burst, max_concurrency, timeout_ms, headers, default_content_type, rate_class, created, updated
		  FROM servers
		  ORDER BY id
		  LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []types.Server
	for rows.Next() {
		var s types.Server
		if err := rows.Scan(
			&s.ID, &s.UID, &s.Name, &s.Username, &s.Password, &s.AuthToken, &s.IPAddress, &s.URL, &s.CallbackURL, &s.CCURLs,
			&s.HTTPMethod, &s.AuthMethod, &s.SystemType, &s.EndpointType, &s.URLParams, &s.AllowCallbacks,
			&s.AllowCopies, &s.UseAsync, &s.UseSSL, &s.ParseResponses, &s.SSLClientCertKeyFile,
			&s.StartSubmissionPeriod, &s.EndSubmissionPeriod, &s.XMLResponseXPath, &s.JSONResponseXPath, &s.Suspended,
			&s.RPS, &s.Burst, &s.MaxConcurrency, &s.TimeoutMS, &s.Headers, &s.DefaultContentType, &s.RateClass, &s.Created, &s.Updated,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ServerRepo) Get(ctx context.Context, id int32) (*types.Server, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, uid, name, username, password, auth_token, ipaddress, url, callback_url, cc_urls,
		       http_method, auth_method, system_type, endpoint_type, url_params, allow_callbacks,
		       allow_copies, use_async, use_ssl, parse_responses, ssl_client_certkey_file,
		       start_submission_period, end_submission_period, xml_response_xpath, json_response_xpath, suspended,
		       rps, burst, max_concurrency, timeout_ms, headers, default_content_type, rate_class, created, updated
		  FROM servers WHERE id=$1`, id)

	var s types.Server
	if err := row.Scan(
		&s.ID, &s.UID, &s.Name, &s.Username, &s.Password, &s.AuthToken, &s.IPAddress, &s.URL, &s.CallbackURL, &s.CCURLs,
		&s.HTTPMethod, &s.AuthMethod, &s.SystemType, &s.EndpointType, &s.URLParams, &s.AllowCallbacks,
		&s.AllowCopies, &s.UseAsync, &s.UseSSL, &s.ParseResponses, &s.SSLClientCertKeyFile,
		&s.StartSubmissionPeriod, &s.EndSubmissionPeriod, &s.XMLResponseXPath, &s.JSONResponseXPath, &s.Suspended,
		&s.RPS, &s.Burst, &s.MaxConcurrency, &s.TimeoutMS, &s.Headers, &s.DefaultContentType, &s.RateClass, &s.Created, &s.Updated,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	return &s, nil
}

func (r *ServerRepo) Create(ctx context.Context, s *types.Server) (*types.Server, error) {
	row := r.db.QueryRow(ctx, `
	INSERT INTO servers (name, username, password, auth_token, ipaddress, url, callback_url, cc_urls,
		http_method, auth_method, system_type, endpoint_type, url_params, allow_callbacks, allow_copies,
		use_async, use_ssl, parse_responses, ssl_client_certkey_file, start_submission_period, end_submission_period,
		xml_response_xpath, json_response_xpath, suspended, rps, burst, max_concurrency, timeout_ms, headers,
		default_content_type, rate_class)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
	        $9,$10,$11,$12,$13,$14,$15,
	        $16,$17,$18,$19,$20,$21,
	        $22,$23,$24,$25,$26,$27,$28,$29,
	        $30,$31)
	RETURNING id, uid, created, updated`,
		s.Name, s.Username, s.Password, s.AuthToken, s.IPAddress, s.URL, s.CallbackURL, s.CCURLs,
		s.HTTPMethod, s.AuthMethod, s.SystemType, s.EndpointType, s.URLParams, s.AllowCallbacks, s.AllowCopies,
		s.UseAsync, s.UseSSL, s.ParseResponses, s.SSLClientCertKeyFile, s.StartSubmissionPeriod, s.EndSubmissionPeriod,
		s.XMLResponseXPath, s.JSONResponseXPath, s.Suspended, s.RPS, s.Burst, s.MaxConcurrency, s.TimeoutMS, s.Headers,
		s.DefaultContentType, s.RateClass,
	)
	now := time.Now()
	s.Created, s.Updated = &now, &now
	if err := row.Scan(&s.ID, &s.UID, &s.Created, &s.Updated); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *ServerRepo) Update(ctx context.Context, s *types.Server) (*types.Server, error) {
	row := r.db.QueryRow(ctx, `
	UPDATE servers SET
		name=$1, username=$2, password=$3, auth_token=$4, ipaddress=$5, url=$6, callback_url=$7, cc_urls=$8,
		http_method=$9, auth_method=$10, system_type=$11, endpoint_type=$12, url_params=$13, allow_callbacks=$14,
		allow_copies=$15, use_async=$16, use_ssl=$17, parse_responses=$18, ssl_client_certkey_file=$19,
		start_submission_period=$20, end_submission_period=$21, xml_response_xpath=$22, json_response_xpath=$23,
		suspended=$24, rps=$25, burst=$26, max_concurrency=$27, timeout_ms=$28, headers=$29,
		default_content_type=$30, rate_class=$31, updated=now()
	WHERE id=$32
	RETURNING updated`,
		s.Name, s.Username, s.Password, s.AuthToken, s.IPAddress, s.URL, s.CallbackURL, s.CCURLs,
		s.HTTPMethod, s.AuthMethod, s.SystemType, s.EndpointType, s.URLParams, s.AllowCallbacks,
		s.AllowCopies, s.UseAsync, s.UseSSL, s.ParseResponses, s.SSLClientCertKeyFile,
		s.StartSubmissionPeriod, s.EndSubmissionPeriod, s.XMLResponseXPath, s.JSONResponseXPath,
		s.Suspended, s.RPS, s.Burst, s.MaxConcurrency, s.TimeoutMS, s.Headers,
		s.DefaultContentType, s.RateClass, s.ID,
	)
	if err := row.Scan(&s.Updated); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *ServerRepo) Delete(ctx context.Context, id int32) error {
	_, err := r.db.Exec(ctx, `DELETE FROM servers WHERE id=$1`, id)
	return err
}
