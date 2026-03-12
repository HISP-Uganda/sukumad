package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"sukumad/config"
)

// --------------------------- SQL -----------------------------

const sqlClaimDeliveries = `
SELECT
  delivery_id, request_id, server_id, method, final_url,
  body, content_type, headers,
  auth_method, username, password, auth_token,
  timeout_ms, rate_key, rps, burst, max_concurrency, server_use_async
FROM claim_deliveries($1)
`
const sqlClaimAsyncPolls = `
SELECT
  delivery_id, server_id, poll_url,
  auth_method, username, password, auth_token,
  timeout_ms, rate_key, rps, burst, max_concurrency
FROM claim_async_polls($1)
`
const sqlStartAsync = `SELECT start_async_delivery($1,$2,$3,$4,$5);`
const sqlFinalize = `SELECT finalize_delivery($1,$2,$3,$4,$5,$6,$7,$8,$9);`
const sqlUpdateAsync = `SELECT update_async_status($1,$2,$3,$4,$5,$6);`

// ------------------------- Types -----------------------------

type DeliveryRow struct {
	DeliveryID     int64
	RequestID      int64
	ServerID       int32
	Method         string
	FinalURL       string
	Body           string
	ContentType    string
	HeadersJSON    []byte
	AuthMethod     string
	Username       *string
	Password       *string
	AuthToken      *string
	TimeoutMs      *int32
	RateKey        string
	RPS            *float64
	Burst          *int32
	MaxConc        *int32
	ServerUseAsync bool
}

type PollRow struct {
	DeliveryID int64
	ServerID   int32
	PollURL    string
	AuthMethod string
	Username   *string
	Password   *string
	AuthToken  *string
	TimeoutMs  *int32
	RateKey    string
	RPS        *float64
	Burst      *int32
	MaxConc    *int32
}

// -------------------- Limiters & Concurrency ------------------

// LimiterSet manages rate limiting and optional concurrency limits at two levels:
//   - Global: a single process-wide limiter applied to every operation.
//   - Per-key: a limiter keyed by an arbitrary string (e.g., target server or route),
//     with an optional bounded semaphore to cap concurrent in-flight operations per key.
//
// The per-key limiter controls requests-per-second (RPS) and burst, while the optional
// semaphore (conc) enforces a maximum number of simultaneous operations for that key.
// Access to internal maps is guarded by mu.
//
// Usage pattern:
//
//	release, err := ls.wait(ctx, key, rps, burst, maxConc)
//	if err != nil { /* context canceled or limiter wait error */ }
//	defer release() // Important: release the per-key concurrency slot if one was acquired.
//	// ... perform the work ...
//
// Note: If maxConc is nil or <= 0, only rate limiting is applied; no concurrency slot is taken.
// If a concurrency slot is taken, release() must be called to free it.
//
// Thread-safety: Safe for concurrent use by multiple goroutines.
//
// See: golang.org/x/time/rate for rate limiting primitives.
type LimiterSet struct {
	Global *rate.Limiter
	mu     sync.Mutex
	per    map[string]*rate.Limiter
	conc   map[string]chan struct{}
}

// NewLimiterSet creates a LimiterSet with a configured global rate limiter.
//   - globalRPS: allowed average requests per second across the entire process.
//   - globalBurst: maximum burst size for the global limiter.
//
// The per-key maps are initialized lazily when a key is first used.
func NewLimiterSet(globalRPS float64, globalBurst int) *LimiterSet {
	return &LimiterSet{
		Global: rate.NewLimiter(rate.Limit(globalRPS), globalBurst),
		per:    make(map[string]*rate.Limiter),
		conc:   make(map[string]chan struct{}),
	}
}

func (ls *LimiterSet) wait(ctx context.Context, key string, rps float64, burst int, maxConc *int32) (release func(), err error) {
	if err := ls.Global.Wait(ctx); err != nil {
		return func() {}, err
	}
	ls.mu.Lock()
	lim := ls.per[key]
	if lim == nil {
		lim = rate.NewLimiter(rate.Limit(rps), burst)
		ls.per[key] = lim
	}
	var sem chan struct{}
	if maxConc != nil && *maxConc > 0 {
		sem = ls.conc[key]
		if sem == nil {
			sem = make(chan struct{}, *maxConc)
			ls.conc[key] = sem
		}
	}
	ls.mu.Unlock()

	if err := lim.Wait(ctx); err != nil {
		return func() {}, err
	}
	if sem != nil {
		select {
		case sem <- struct{}{}:
			return func() { <-sem }, nil
		case <-ctx.Done():
			return func() {}, ctx.Err()
		}
	}
	return func() {}, nil
}

// ------------------------ Logger setup ------------------------

func setupLogger(cfg config.Config) {
	// Level
	level := log.InfoLevel
	if cfg.Debug {
		level = log.DebugLevel
	}
	if v := cfg.LogLevel; v != "" {
		if parsed, err := log.ParseLevel(strings.ToLower(v)); err == nil {
			level = parsed
		}
	}
	log.SetLevel(level)

	// Format (default JSON for structured logs; text if explicitly set)
	format := strings.ToLower(cfg.LogFormat)
	if format == "text" {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	} else {
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	// Include PID to help correlate multi-process deployments
	log.WithField("pid", os.Getpid()).Info("logger initialized")
}

// ----------------------------- Main ---------------------------

func main() {
	cfg := config.AppConfig
	setupLogger(cfg)

	if cfg.DBURL == "" {
		log.Fatal("SUKUMAD_DBURL (or dburl in YAML) is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		log.WithError(err).Fatal("pgxpool connect failed")
	}
	defer pool.Close()

	limiters := NewLimiterSet(cfg.GlobalRPS, cfg.GlobalBurst)

	sendCh := make(chan DeliveryRow, cfg.ClaimBatch*2)
	pollCh := make(chan PollRow, cfg.ClaimBatch*2)

	var wg sync.WaitGroup
	wg.Add(cfg.SendWorkers)
	for i := 0; i < cfg.SendWorkers; i++ {
		go func(id int) {
			defer wg.Done()
			sendWorker(ctx, pool, limiters, cfg, sendCh, id)
		}(i + 1)
	}
	wg.Add(cfg.PollWorkers)
	for i := 0; i < cfg.PollWorkers; i++ {
		go func(id int) {
			defer wg.Done()
			pollWorker(ctx, pool, limiters, cfg, pollCh, id)
		}(i + 1)
	}

	ticker := time.NewTicker(cfg.ClaimInterval)
	defer ticker.Stop()

	log.WithFields(log.Fields{
		"claim_batch":  cfg.ClaimBatch,
		"interval":     cfg.ClaimInterval.String(),
		"send_workers": cfg.SendWorkers,
		"poll_workers": cfg.PollWorkers,
		"global_rps":   cfg.GlobalRPS,
		"global_burst": cfg.GlobalBurst,
	}).Info("worker started")

	for {
		select {
		case <-ctx.Done():
			close(sendCh)
			close(pollCh)
			wg.Wait()
			log.Info("graceful shutdown complete")
			return
		case <-ticker.C:
			claimAndDispatch(ctx, pool, cfg.ClaimBatch, sendCh, pollCh)
		}
	}
}

// -------------------------- Claimers --------------------------

func claimAndDispatch(ctx context.Context, pool *pgxpool.Pool, batch int, sendCh chan<- DeliveryRow, pollCh chan<- PollRow) {
	// initial sends
	start := time.Now()
	rows, err := pool.Query(ctx, sqlClaimDeliveries, batch)
	if err != nil {
		log.WithError(err).Error("claim_deliveries failed")
	} else {
		count := 0
		for rows.Next() {
			var r DeliveryRow
			if err := rows.Scan(
				&r.DeliveryID, &r.RequestID, &r.ServerID, &r.Method, &r.FinalURL,
				&r.Body, &r.ContentType, &r.HeadersJSON,
				&r.AuthMethod, &r.Username, &r.Password, &r.AuthToken,
				&r.TimeoutMs, &r.RateKey, &r.RPS, &r.Burst, &r.MaxConc, &r.ServerUseAsync,
			); err != nil {
				log.WithError(err).Error("scan claim_deliveries row")
				continue
			}
			count++
			select {
			case sendCh <- r:
			case <-ctx.Done():
				rows.Close()
				return
			}
		}
		rows.Close()
		if count > 0 {
			log.WithFields(log.Fields{"claimed": count, "took": time.Since(start).String()}).Info("claimed deliveries")
		} else {
			log.Debug("no deliveries ready")
		}
	}

	// async polls
	start = time.Now()
	prows, err := pool.Query(ctx, sqlClaimAsyncPolls, batch)
	if err != nil {
		log.WithError(err).Error("claim_async_polls failed")
		return
	}
	pcount := 0
	for prows.Next() {
		var p PollRow
		if err := prows.Scan(
			&p.DeliveryID, &p.ServerID, &p.PollURL, &p.AuthMethod,
			&p.Username, &p.Password, &p.AuthToken, &p.TimeoutMs,
			&p.RateKey, &p.RPS, &p.Burst, &p.MaxConc,
		); err != nil {
			log.WithError(err).Error("scan claim_async_polls row")
			continue
		}
		pcount++
		select {
		case pollCh <- p:
		case <-ctx.Done():
			prows.Close()
			return
		}
	}
	prows.Close()
	if pcount > 0 {
		log.WithFields(log.Fields{"claimed": pcount, "took": time.Since(start).String()}).Info("claimed async polls")
	} else {
		log.Debug("no async polls ready")
	}
}

// --------------------------- Workers --------------------------

func sendWorker(ctx context.Context, pool *pgxpool.Pool, ls *LimiterSet, cfg config.Config, in <-chan DeliveryRow, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case row, ok := <-in:
			if !ok {
				return
			}
			rps := derefFloat(row.RPS, 1.0)
			burst := int(derefInt32(row.Burst, int32(rps)))
			release, err := ls.wait(ctx, row.RateKey, rps, burst, row.MaxConc)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"worker": workerID, "rate_key": row.RateKey,
				}).Warn("limiter wait error")
				continue
			}
			if err := handleSendResty(ctx, pool, cfg, row, workerID); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"worker": workerID, "delivery_id": row.DeliveryID,
				}).Error("send failed")
			}
			release()
		}
	}
}

func pollWorker(ctx context.Context, pool *pgxpool.Pool, ls *LimiterSet, cfg config.Config, in <-chan PollRow, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case row, ok := <-in:
			if !ok {
				return
			}
			rps := derefFloat(row.RPS, 1.0)
			burst := int(derefInt32(row.Burst, int32(rps)))
			release, err := ls.wait(ctx, row.RateKey, rps, burst, row.MaxConc)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{
					"worker": workerID, "rate_key": row.RateKey,
				}).Warn("limiter wait error")
				continue
			}
			if err := handlePollResty(ctx, pool, cfg, row, workerID); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"worker": workerID, "delivery_id": row.DeliveryID,
				}).Error("poll failed")
			}
			release()
		}
	}
}

// ----------------------- Resty helpers ------------------------

func restyClientFor(timeoutMs *int32, defaultTimeout time.Duration) *resty.Client {
	t := defaultTimeout
	if timeoutMs != nil && *timeoutMs > 0 {
		t = time.Duration(*timeoutMs) * time.Millisecond
	}
	c := resty.New().SetTimeout(t)

	// Tune transport if you like (keep defaults for brevity)

	return c
}

func attachRestyLogging(c *resty.Client, label string) {
	// Debug-only verbose timing; bodies are NOT logged.
	c.OnBeforeRequest(func(cli *resty.Client, req *resty.Request) error {
		if log.IsLevelEnabled(log.DebugLevel) {
			log.WithFields(log.Fields{
				"phase":   "before_request",
				"label":   label,
				"method":  req.Method,
				"url":     req.URL,
				"headers": len(req.Header),
				"body_sz": bodySize(req.Body),
			}).Debug("http")
		}
		return nil
	})
	c.OnAfterResponse(func(cli *resty.Client, resp *resty.Response) error {
		if log.IsLevelEnabled(log.DebugLevel) {
			log.WithFields(log.Fields{
				"phase":    "after_response",
				"label":    label,
				"method":   resp.Request.Method,
				"url":      resp.Request.URL,
				"status":   resp.StatusCode(),
				"took_ms":  resp.Time().Milliseconds(),
				"hdr_recv": len(resp.Header()),
				"body_sz":  len(resp.Body()),
			}).Debug("http")
		}
		return nil
	})
}

func buildRestyRequest(
	ctx context.Context,
	c *resty.Client,
	contentType, body string,
	headersJSON []byte,
	authMethod string,
	username, password, token *string,
) *resty.Request {
	r := c.R().SetContext(ctx)

	// headers
	if contentType != "" {
		r.SetHeader("Content-Type", contentType)
	}
	if len(headersJSON) > 0 {
		var hdr map[string]any
		if json.Unmarshal(headersJSON, &hdr) == nil {
			h := map[string]string{}
			for k, v := range hdr {
				h[k] = toString(v)
			}
			r.SetHeaders(h)
		}
	}

	// auth
	switch strings.ToLower(authMethod) {
	case "basic":
		if username != nil {
			r.SetBasicAuth(*username, deref(password))
		}
	case "bearer":
		if token != nil && *token != "" {
			r.SetAuthToken(*token)
		}
	case "token":
		if token != nil && *token != "" {
			r.SetHeader("Authorization", "Token "+*token)
		}

	}

	// body
	if body != "" {
		r.SetBody([]byte(body))
	}

	return r
}

func parseRetryAfterStr(s string) *time.Time {
	if s == "" {
		return nil
	}
	if secs, err := strconv.Atoi(s); err == nil {
		t := time.Now().Add(time.Duration(secs) * time.Second)
		return &t
	}
	if t, err := httpParseTime(s); err == nil {
		return &t
	}
	return nil
}

// --------------------- Handlers (Resty) -----------------------

func handleSendResty(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, row DeliveryRow, workerID int) error {
	client := restyClientFor(row.TimeoutMs, cfg.DownstreamClientTimeout)
	attachRestyLogging(client, "send")
	req := buildRestyRequest(ctx, client, row.ContentType, row.Body, row.HeadersJSON, row.AuthMethod, row.Username, row.Password, row.AuthToken)

	start := time.Now()
	resp, err := req.Execute(row.Method, row.FinalURL)
	dur := time.Since(start)

	e := log.WithFields(log.Fields{
		"worker":     workerID,
		"delivery":   row.DeliveryID,
		"server_id":  row.ServerID,
		"rate_key":   row.RateKey,
		"method":     row.Method,
		"url":        row.FinalURL,
		"durationMs": dur.Milliseconds(),
	})

	if err != nil {
		e.WithError(err).Warn("request error; scheduling retry")
		return finalize(pool, row.DeliveryID, false, 0, "", err, nil)
	}

	status := resp.StatusCode()
	respBytes := resp.Body()
	respStr := string(respBytes)

	switch {
	case row.ServerUseAsync && status == 202:
		loc := resp.Header().Get("Location")
		jobID, pollURL := extractAsyncHandlesFromLocation(loc)
		e = e.WithFields(log.Fields{"status": status, "async_location": loc, "job_id": jobID})
		if jobID == "" {
			e.Warn("accepted but no async job id; will retry")
			return finalize(pool, row.DeliveryID, false, status, truncate(respStr, 1024), errors.New("missing async job id"), parseRetryAfterStr(resp.Header().Get("Retry-After")))
		}
		if err := startAsync(pool, row.DeliveryID, jobID, pollURL, "queued", cfg.PollAgainSeconds); err != nil {
			e.WithError(err).Error("start_async_delivery failed")
			return err
		}
		e.Info("async job started")
		return nil

	case status >= 200 && status < 300:
		e = e.WithField("status", status)
		e.Info("delivery success")
		return finalize(pool, row.DeliveryID, true, status, truncate(respStr, 4096), nil, nil)

	case status == 429 || status >= 500:
		e = e.WithField("status", status)
		e.Warn("retryable response")
		return finalize(pool, row.DeliveryID, false, status, truncate(respStr, 4096), errors.New("retryable"), parseRetryAfterStr(resp.Header().Get("Retry-After")))

	default:
		e = e.WithField("status", status)
		e.Error("terminal failure")
		return finalize(pool, row.DeliveryID, false, status, truncate(respStr, 4096), errors.New("terminal"), nil)
	}
}

func handlePollResty(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, row PollRow, workerID int) error {
	client := restyClientFor(row.TimeoutMs, cfg.DownstreamClientTimeout)
	attachRestyLogging(client, "poll")
	req := buildRestyRequest(ctx, client, "", "", nil, row.AuthMethod, row.Username, row.Password, row.AuthToken)

	start := time.Now()
	resp, err := req.Get(row.PollURL)
	dur := time.Since(start)

	e := log.WithFields(log.Fields{
		"worker":     workerID,
		"delivery":   row.DeliveryID,
		"server_id":  row.ServerID,
		"rate_key":   row.RateKey,
		"url":        row.PollURL,
		"durationMs": dur.Milliseconds(),
	})

	if err != nil {
		e.WithError(err).Warn("poll error; will re-poll")
		return updateAsync(pool, row.DeliveryID, "RUNNING", map[string]any{"error": err.Error()}, cfg.PollAgainSeconds)
	}

	b := resp.Body()
	status, succ, fail := mapDhis2Job(b)
	switch {
	case succ != "":
		e.WithField("status", succ).Info("async success")
		return updateAsync(pool, row.DeliveryID, succ, jsonRawToMap(b), 0)
	case fail != "":
		e.WithField("status", fail).Warn("async failed")
		return updateAsync(pool, row.DeliveryID, fail, jsonRawToMap(b), 0)
	default:
		e.WithField("status", status).Debug("async still running")
		return updateAsync(pool, row.DeliveryID, status, jsonRawToMap(b), cfg.PollAgainSeconds)
	}
}

// --------------------- DB wrappers & utils --------------------

func startAsync(pool *pgxpool.Pool, deliveryID int64, jobID, pollURL, initial string, firstPollDelaySeconds int) error {
	_, err := pool.Exec(context.Background(), sqlStartAsync, deliveryID, jobID, pollURL, initial, firstPollDelaySeconds)
	return err
}

func finalize(pool *pgxpool.Pool, deliveryID int64, ok bool, statusCode int, responseBody string, cause error, retryAfter *time.Time) error {
	errMsg := ""
	if cause != nil {
		errMsg = cause.Error()
	}
	var retryAt any
	if retryAfter != nil {
		retryAt = *retryAfter
	} else {
		retryAt = nil
	}
	_, err := pool.Exec(context.Background(), sqlFinalize, deliveryID, ok, statusCode, responseBody, errMsg, retryAt, 8, 2, 120)
	return err
}

func updateAsync(pool *pgxpool.Pool, deliveryID int64, status string, payload map[string]any, pollAgainSeconds int) error {
	var jb []byte
	if payload != nil {
		jb, _ = json.Marshal(payload)
	}
	_, err := pool.Exec(context.Background(), sqlUpdateAsync, deliveryID, status, jb, []string{"COMPLETED", "SUCCESS", "OK"}, []string{"FAILED", "ERROR"}, pollAgainSeconds)
	return err
}

// ----------------------------- Helpers ------------------------

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(collapseSpace(t))

	case []any:
		if len(t) == 0 {
			return ""
		}
		var b strings.Builder
		for i, vv := range t {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(toString(vv))
		}
		return b.String()

	case fmt.Stringer:
		return strings.TrimSpace(collapseSpace(t.String()))

	default:
		// Try JSON for complex types (maps/structs), fall back to fmt.Sprint.
		if b, err := json.Marshal(v); err == nil {
			return strings.TrimSpace(collapseSpace(string(b)))
		}
		return strings.TrimSpace(collapseSpace(fmt.Sprint(v)))
	}
}

func extractAsyncHandlesFromLocation(loc string) (jobID, pollURL string) {
	if loc == "" {
		return "", ""
	}
	parts := strings.Split(strings.TrimRight(loc, "/"), "/")
	return parts[len(parts)-1], loc
}

func mapDhis2Job(b []byte) (status, terminalSuccess, terminalFail string) {
	var obj map[string]any
	if json.Unmarshal(b, &obj) != nil {
		return "RUNNING", "", ""
	}
	if s, ok := obj["status"].(string); ok {
		u := strings.ToUpper(s)
		switch u {
		case "COMPLETED", "SUCCESS", "OK":
			return "", "COMPLETED", ""
		case "FAILED", "ERROR":
			return "", "", "FAILED"
		case "RUNNING", "QUEUED":
			return u, "", ""
		}
	}
	return "RUNNING", "", ""
}

func jsonRawToMap(b []byte) map[string]any {
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func bodySize(body any) int {
	switch v := body.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	default:
		return 0
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
func derefFloat(p *float64, def float64) float64 {
	if p == nil || *p <= 0 {
		return def
	}
	return *p
}
func derefInt32(p *int32, def int32) int32 {
	if p == nil || *p <= 0 {
		return def
	}
	return *p
}

// Lightweight HTTP-date parser (avoid importing net/http for one function)
func httpParseTime(v string) (time.Time, error) {
	// net/http uses http.ParseTime with multiple layouts; here we try common RFC1123
	layouts := []string{time.RFC1123, time.RFC1123Z, time.RFC850, time.ANSIC, time.RFC822, time.RFC822Z}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("parse time failed")
}
