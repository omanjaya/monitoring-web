package mysql

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
)

type DNSRepository struct {
	db *sqlx.DB
}

func NewDNSRepository(db *sqlx.DB) *DNSRepository {
	return &DNSRepository{db: db}
}

// SaveDNSScan saves a DNS scan record to the database
func (r *DNSRepository) SaveDNSScan(ctx context.Context, scan *domain.DNSScanRecord) error {
	scan.PrepareJSON()

	query := `
		INSERT INTO dns_scans (website_id, domain_name, has_spf, has_dmarc, spf_record, dmarc_record,
			nameservers, mx_records, dns_records, subdomains, subdomain_count, scan_duration_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		scan.WebsiteID,
		scan.DomainName,
		scan.HasSPF,
		scan.HasDMARC,
		scan.SPFRecord,
		scan.DMARCRecord,
		scan.NameserversJSON,
		scan.MXRecordsJSON,
		scan.DNSRecordsJSON,
		scan.SubdomainsJSON,
		scan.SubdomainCount,
		scan.ScanDurationMs,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	scan.ID = id
	return nil
}

// GetLatestByWebsite returns the latest DNS scan for a website ID
func (r *DNSRepository) GetLatestByWebsite(ctx context.Context, websiteID int64) (*domain.DNSScanRecord, error) {
	var scan domain.DNSScanRecord
	query := `SELECT * FROM dns_scans WHERE website_id = ? ORDER BY created_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &scan, query, websiteID)
	if err != nil {
		return nil, err
	}
	scan.ParseJSON()
	return &scan, nil
}

// GetLatestByDomain returns the latest DNS scan for a domain name
func (r *DNSRepository) GetLatestByDomain(ctx context.Context, domainName string) (*domain.DNSScanRecord, error) {
	var scan domain.DNSScanRecord
	query := `SELECT * FROM dns_scans WHERE domain_name = ? ORDER BY created_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &scan, query, domainName)
	if err != nil {
		return nil, err
	}
	scan.ParseJSON()
	return &scan, nil
}

// GetAll returns the latest DNS scan per domain
func (r *DNSRepository) GetAll(ctx context.Context) ([]domain.DNSScanRecord, error) {
	query := `
		SELECT d.* FROM dns_scans d
		INNER JOIN (
			SELECT domain_name, MAX(id) AS max_id
			FROM dns_scans
			GROUP BY domain_name
		) latest ON d.id = latest.max_id
		ORDER BY d.domain_name
	`
	var scans []domain.DNSScanRecord
	err := r.db.SelectContext(ctx, &scans, query)
	if err != nil {
		return nil, err
	}
	for i := range scans {
		scans[i].ParseJSON()
	}
	return scans, nil
}
