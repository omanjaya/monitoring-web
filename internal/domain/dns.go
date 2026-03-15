package domain

import (
	"encoding/json"
	"time"
)

// DNSRecord represents a single DNS record
type DNSRecord struct {
	Type  string `json:"type"`  // A, AAAA, CNAME, MX, NS, TXT, SOA
	Name  string `json:"name"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// SubdomainResult represents a discovered subdomain
type SubdomainResult struct {
	Subdomain  string    `json:"subdomain"`
	IP         string    `json:"ip,omitempty"`
	StatusCode int       `json:"status_code,omitempty"`
	Title      string    `json:"title,omitempty"`
	FoundAt    time.Time `json:"found_at"`
	Source     string    `json:"source"` // dns, bruteforce, certificate
}

// DNSScanRecord represents a DNS scan result stored in the database
type DNSScanRecord struct {
	ID             int64     `db:"id" json:"id"`
	WebsiteID      int64     `db:"website_id" json:"website_id"`
	DomainName     string    `db:"domain_name" json:"domain_name"`
	HasSPF         bool      `db:"has_spf" json:"has_spf"`
	HasDMARC       bool      `db:"has_dmarc" json:"has_dmarc"`
	SPFRecord      string    `db:"spf_record" json:"spf_record"`
	DMARCRecord    string    `db:"dmarc_record" json:"dmarc_record"`
	Nameservers    []string          `db:"-" json:"nameservers"`
	MXRecords      []string          `db:"-" json:"mx_records"`
	DNSRecords     []DNSRecord       `db:"-" json:"dns_records"`
	Subdomains     []SubdomainResult `db:"-" json:"subdomains"`
	SubdomainCount int       `db:"subdomain_count" json:"subdomain_count"`
	ScanDurationMs int       `db:"scan_duration_ms" json:"scan_duration_ms"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	// Raw JSON for DB scanning
	NameserversJSON []byte `db:"nameservers" json:"-"`
	MXRecordsJSON   []byte `db:"mx_records" json:"-"`
	DNSRecordsJSON  []byte `db:"dns_records" json:"-"`
	SubdomainsJSON  []byte `db:"subdomains" json:"-"`
}

// ParseJSON unmarshals the JSON byte fields into the typed fields
func (r *DNSScanRecord) ParseJSON() {
	if r.NameserversJSON != nil {
		_ = json.Unmarshal(r.NameserversJSON, &r.Nameservers)
	}
	if r.MXRecordsJSON != nil {
		_ = json.Unmarshal(r.MXRecordsJSON, &r.MXRecords)
	}
	if r.DNSRecordsJSON != nil {
		_ = json.Unmarshal(r.DNSRecordsJSON, &r.DNSRecords)
	}
	if r.SubdomainsJSON != nil {
		_ = json.Unmarshal(r.SubdomainsJSON, &r.Subdomains)
	}
}

// PrepareJSON marshals typed fields to JSON bytes for saving
func (r *DNSScanRecord) PrepareJSON() {
	r.NameserversJSON, _ = json.Marshal(r.Nameservers)
	r.MXRecordsJSON, _ = json.Marshal(r.MXRecords)
	r.DNSRecordsJSON, _ = json.Marshal(r.DNSRecords)
	r.SubdomainsJSON, _ = json.Marshal(r.Subdomains)
}

// DNSScanResult contains the full result of a DNS scan
type DNSScanResult struct {
	WebsiteID    int64             `json:"website_id"`
	Domain       string            `json:"domain"`
	Records      []DNSRecord       `json:"records"`
	Subdomains   []SubdomainResult `json:"subdomains"`
	Nameservers  []string          `json:"nameservers"`
	MXRecords    []string          `json:"mx_records"`
	SPFRecord    string            `json:"spf_record,omitempty"`
	DMARCRecord  string            `json:"dmarc_record,omitempty"`
	ScannedAt    time.Time         `json:"scanned_at"`
	ScanDuration int               `json:"scan_duration_ms"`
}
