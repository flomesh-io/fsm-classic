package route

import gwv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

type ConfigSpec struct {
	Listeners             []Listener               `json:"Listeners"`
	Certificate           *Certificate             `json:"Certificate"`
	HTTPRouteRules        map[string]HTTPRouteRule `json:"HTTPRouteRules"`
	PassthroughRouteRules *PassthroughRouteRules   `json:"PassthroughRouteRules"`
	TCPRouteRules         map[string]string        `json:"TCPRouteRules"`
	UDPPRouteRules        map[string]string        `json:"UDPPRouteRules"`
	Services              map[string]ServiceConfig `json:"Services"`
	Chains                Chains                   `json:"Chains"`
	Features              Features                 `json:"Features"`
}

type Listener struct {
	Protocol string `json:"Protocol"`
	Port     int    `json:"Port"`
	TLS      *TLS   `json:"TLS,omitempty"`
}

type TLS struct {
	TLSModeType  string        `json:"TLSModeType"`
	MTLS         bool          `json:"mTLS"`
	Certificates []Certificate `json:"Certificates"`
}

type Certificate struct {
	CommonName   string `json:"CommonName"`
	SerialNumber string `json:"SerialNumber"`
	Expiration   string `json:"Expiration"`
	CertChain    string `json:"CertChain"`
	PrivateKey   string `json:"PrivateKey"`
	IssuingCA    string `json:"IssuingCA"`
}

type HTTPRouteRule struct {
	Matches   []TrafficMatch `json:"Matches"`
	RateLimit *RateLimit     `json:"RateLimit"`
}

type TrafficMatch struct {
	Path          string            `json:"Path"`
	Type          string            `json:"Type"`
	Headers       map[string]string `json:"Headers"`
	Methods       []string          `json:"Methods"`
	TargetService map[string]int    `json:"TargetService"`
	RateLimit     *RateLimit        `json:"RateLimit,omitempty"`
}

type RateLimit struct {
	Backlog              int             `json:"Backlog"`
	Requests             int             `json:"Requests"`
	Burst                int             `json:"Burst"`
	StatTimeWindow       int             `json:"StatTimeWindow"`
	ResponseStatusCode   int             `json:"ResponseStatusCode"`
	ResponseHeadersToAdd []NameValuePair `json:"ResponseHeadersToAdd"`
}

type NameValuePair struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type PassthroughRouteRules struct {
	DefaultUpstreamPort int                                `json:"DefaultUpstreamPort"`
	Mappings            map[string]PassthroughRouteMapping `json:"Mappings"`
}

type PassthroughRouteMapping map[string]string

type ServiceConfig struct {
	Endpoints          map[string]Endpoint `json:"Endpoints"`
	Filters            []Filter            `json:"Filters"`
	ConnectionSettings *ConnectionSettings `json:"ConnectionSettings"`
	RetryPolicy        *RetryPolicy        `json:"RetryPolicy"`
	UpstreamCert       *UpstreamCert       `json:"UpstreamCert"`
}

type Endpoint struct {
	Weight       int               `json:"Weight"`
	Tags         map[string]string `json:"Tags"`
	UpstreamCert *UpstreamCert     `json:"UpstreamCert"`
}

type Filter gwv1beta1.HTTPRouteFilter

type ConnectionSettings struct {
	TCP  *TCPConnectionSettings  `json:"tcp"`
	HTTP *HTTPConnectionSettings `json:"http"`
}

type TCPConnectionSettings struct {
	MaxConnections int `json:"MaxConnections"`
}

type HTTPConnectionSettings struct {
	MaxRequestsPerConnection int             `json:"MaxRequestsPerConnection"`
	MaxPendingRequests       int             `json:"MaxPendingRequests"`
	CircuitBreaker           *CircuitBreaker `json:"CircuitBreaker"`
}

type CircuitBreaker struct {
	MinRequestAmount        int     `json:"MinRequestAmount"`
	StatTimeWindow          int     `json:"StatTimeWindow"`
	SlowTimeThreshold       float64 `json:"SlowTimeThreshold"`
	SlowAmountThreshold     int     `json:"SlowAmountThreshold"`
	SlowRatioThreshold      float64 `json:"SlowRatioThreshold"`
	ErrorAmountThreshold    int     `json:"ErrorAmountThreshold"`
	ErrorRatioThreshold     float64 `json:"ErrorRatioThreshold"`
	DegradedTimeWindow      int     `json:"DegradedTimeWindow"`
	DegradedStatusCode      int     `json:"DegradedStatusCode"`
	DegradedResponseContent string  `json:"DegradedResponseContent"`
}
type UpstreamCert struct {
	OsmIssued  bool   `json:"OsmIssued"`
	CertChain  string `json:"CertChain"`
	PrivateKey string `json:"PrivateKey"`
	IssuingCA  string `json:"IssuingCA"`
}

type RetryPolicy struct {
	RetryOn                  string `json:"RetryOn"`
	PerTryTimeout            int    `json:"PerTryTimeout"`
	NumRetries               int    `json:"NumRetries"`
	RetryBackoffBaseInterval int    `json:"RetryBackoffBaseInterval"`
}
type Chains struct {
	InboundHTTP  []string `json:"inbound-http"`
	InboundTCP   []string `json:"inbound-tcp"`
	OutboundHTTP []string `json:"outbound-http"`
	OutboundTCP  []string `json:"outbound-tcp"`
}

type Features struct {
	Logging struct{} `json:"Logging"`
	Tracing struct{} `json:"Tracing"`
	Metrics struct{} `json:"Metrics"`
}
