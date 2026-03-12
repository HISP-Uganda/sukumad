package types

import "time"

// APIError response for swagger
type APIError struct {
	Error string `json:"error" example:"bad request"`
}

type Server struct {
	ID                    int32    `json:"id" example:"1"`
	UID                   *string  `json:"uid,omitempty" example:"9e2b34f0-1fb2-4a2f-a6d9-0b2a1f92d144"`
	Name                  string   `json:"name" binding:"required" example:"DHIS2 Main"`
	Username              *string  `json:"username,omitempty" example:"admin"`
	Password              *string  `json:"password,omitempty" example:"district"`
	AuthToken             *string  `json:"auth_token,omitempty"`
	IPAddress             *string  `json:"ipaddress,omitempty" example:"10.0.0.12"`
	URL                   string   `json:"url" binding:"required,url" example:"https://dhis2.example.org/api/dataValueSets"`
	CallbackURL           *string  `json:"callback_url,omitempty" example:"https://example.org/callback"`
	CCURLs                []string `json:"cc_urls,omitempty" example:"[\"https://a\",\"https://b\"]"`
	HTTPMethod            string   `json:"http_method" binding:"required,httpmethod" example:"POST"`
	AuthMethod            string   `json:"auth_method" binding:"required,authmethod" example:"basic"`
	SystemType            *string  `json:"system_type,omitempty"`
	EndpointType          *string  `json:"endpoint_type,omitempty"`
	URLParams             any      `json:"url_params,omitempty"`
	AllowCallbacks        *bool    `json:"allow_callbacks,omitempty"`
	AllowCopies           *bool    `json:"allow_copies,omitempty"`
	UseAsync              *bool    `json:"use_async,omitempty" example:"true"`
	UseSSL                *bool    `json:"use_ssl,omitempty" example:"true"`
	ParseResponses        *bool    `json:"parse_responses,omitempty"`
	SSLClientCertKeyFile  *string  `json:"ssl_client_certkey_file,omitempty"`
	StartSubmissionPeriod *int     `json:"start_submission_period,omitempty" example:"0"`
	EndSubmissionPeriod   *int     `json:"end_submission_period,omitempty" example:"2359"`
	XMLResponseXPath      *string  `json:"xml_response_xpath,omitempty"`
	JSONResponseXPath     *string  `json:"json_response_xpath,omitempty"`
	Suspended             *bool    `json:"suspended,omitempty" example:"false"`

	RPS                *float64          `json:"rps,omitempty" example:"10"`
	Burst              *int32            `json:"burst,omitempty" example:"20"`
	MaxConcurrency     *int32            `json:"max_concurrency,omitempty" example:"2"`
	TimeoutMS          *int32            `json:"timeout_ms,omitempty" example:"10000"`
	Headers            map[string]string `json:"headers,omitempty" example:"{\"Accept\":\"application/json\"}"`
	DefaultContentType string            `json:"default_content_type" binding:"required" example:"application/json"`

	RateClass string     `json:"rate_class,omitempty" example:"server:1"`
	Created   *time.Time `json:"created,omitempty"`
	Updated   *time.Time `json:"updated,omitempty"`
}

type Request struct {
	ID          int64   `json:"id" example:"42"`
	UID         *string `json:"uid,omitempty"`
	Source      *int32  `json:"source,omitempty"`
	Destination *int32  `json:"destination,omitempty" example:"1"`
	CCServers   []int32 `json:"cc_servers,omitempty" example:"[3,5]"`
	DependsOn   *int64  `json:"depends_on,omitempty"`
	BatchID     *string `json:"batchid,omitempty"`

	URLSuffix        *string `json:"url_suffix,omitempty" binding:"omitempty,urlsuffix" example:"/summary"`
	Body             string  `json:"body" binding:"required" example:"{\"dataSet\":\"abc\",\"dataValues\":[]}"` // if BodyIsQueryParam, use query string
	BodyIsQueryParam *bool   `json:"body_is_query_param,omitempty" example:"false"`

	FrequencyType *string `json:"frequency_type,omitempty" example:"monthly"`
	Period        *string `json:"period,omitempty" example:"202501"`
	Week          *string `json:"week,omitempty"`
	Month         *string `json:"month,omitempty"`
	Year          *int32  `json:"year,omitempty" example:"2025"`
	MSISDN        *string `json:"msisdn,omitempty"`
	RawMsg        *string `json:"raw_msg,omitempty"`
	Facility      *string `json:"facility,omitempty"`
	District      *string `json:"district,omitempty"`
	ReportType    *string `json:"report_type,omitempty"`
	ObjectType    *string `json:"object_type,omitempty"`
	Extras        any     `json:"extras,omitempty"`

	Suspended      *bool      `json:"suspended,omitempty"`
	IdempotencyKey *string    `json:"idempotency_key,omitempty"`
	Created        *time.Time `json:"created,omitempty"`
	Updated        *time.Time `json:"updated,omitempty"`
}

type Delivery struct {
	ID           int64      `json:"id" example:"1001"`
	RequestID    int64      `json:"request_id" example:"42"`
	ServerID     int32      `json:"server_id" example:"1"`
	Status       string     `json:"status" example:"ready"`
	Attempts     int32      `json:"attempts" example:"0"`
	NextRunAt    time.Time  `json:"next_run_at"`
	RetryAfterAt *time.Time `json:"retry_after_at,omitempty"`
	LastError    *string    `json:"last_error,omitempty"`
	StatusCode   *int32     `json:"status_code,omitempty" example:"200"`
	ResponseBody *string    `json:"response_body,omitempty"`
	Priority     int32      `json:"priority" example:"100"`
	ScheduledAt  time.Time  `json:"scheduled_at"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`

	RateClass     *string  `json:"rate_class,omitempty"`
	RPSOverride   *float64 `json:"rps_override,omitempty"`
	BurstOverride *int32   `json:"burst_override,omitempty"`
	TimeoutMS     *int32   `json:"timeout_ms,omitempty"`
	MaxAttempts   *int32   `json:"max_attempts,omitempty"`

	DependsOnDeliveryID *int64 `json:"depends_on_delivery_id,omitempty"`

	AsyncJobID      *string    `json:"async_jobid,omitempty"`
	AsyncStatus     *string    `json:"async_status,omitempty"`
	AsyncResponse   any        `json:"async_response,omitempty"`
	AsyncPollURL    *string    `json:"async_poll_url,omitempty"`
	AsyncLastPolled *time.Time `json:"async_last_polled,omitempty"`
}
