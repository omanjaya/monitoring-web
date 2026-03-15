-- Migration 014: Add composite indexes for performance
-- MySQL 8.0 compatible (CREATE INDEX without IF NOT EXISTS)

-- Indexes for alerts table
CREATE INDEX idx_alerts_website_type_resolved ON alerts(website_id, type, is_resolved);
CREATE INDEX idx_alerts_resolved_at ON alerts(resolved_at);
CREATE INDEX idx_alerts_created_at ON alerts(created_at);

-- Indexes for dork_detections table
CREATE INDEX idx_dork_detections_website_resolved ON dork_detections(website_id, is_resolved, is_false_positive);
CREATE INDEX idx_dork_detections_scan_id ON dork_detections(scan_id);

-- Indexes for vulnerability_findings table
CREATE INDEX idx_vuln_findings_website_resolved ON vulnerability_findings(website_id, is_resolved);
CREATE INDEX idx_vuln_findings_severity ON vulnerability_findings(severity);

-- Indexes for website_checks table
CREATE INDEX idx_website_checks_website_created ON website_checks(website_id, checked_at);
CREATE INDEX idx_website_checks_status ON website_checks(status);

-- Indexes for dns_records table
CREATE INDEX idx_dns_records_website ON dns_records(website_id);

-- Indexes for security_headers table
CREATE INDEX idx_security_headers_website ON security_headers(website_id);
