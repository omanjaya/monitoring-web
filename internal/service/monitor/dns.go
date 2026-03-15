package monitor

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

// DNSScanner performs DNS record lookups and subdomain enumeration
type DNSScanner struct {
	cfg         *config.Config
	websiteRepo *mysql.WebsiteRepository
	dnsRepo     *mysql.DNSRepository
	httpClient  *http.Client
	resolver    *net.Resolver
}

// NewDNSScanner creates a new DNS scanner instance
func NewDNSScanner(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	dnsRepo *mysql.DNSRepository,
) *DNSScanner {
	return &DNSScanner{
		cfg:         cfg,
		websiteRepo: websiteRepo,
		dnsRepo:     dnsRepo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 10 * time.Second}
				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		},
	}
}

// ScanDNS performs a full DNS scan for a website
func (s *DNSScanner) ScanDNS(ctx context.Context, w *domain.Website) (*domain.DNSScanResult, error) {
	start := time.Now()

	parsedURL, err := url.Parse(w.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	domainName := parsedURL.Hostname()

	result := &domain.DNSScanResult{
		WebsiteID: w.ID,
		Domain:    domainName,
		ScannedAt: time.Now(),
	}

	// 1. Get DNS records
	result.Records = s.getDNSRecords(ctx, domainName)

	// 2. Get nameservers
	result.Nameservers = s.getNameservers(ctx, domainName)

	// 3. Get MX records
	result.MXRecords = s.getMXRecords(ctx, domainName)

	// 4. Check SPF
	result.SPFRecord = s.getSPFRecord(ctx, domainName)

	// 5. Check DMARC
	result.DMARCRecord = s.getDMARCRecord(ctx, domainName)

	// 6. Enumerate subdomains
	result.Subdomains = s.enumerateSubdomains(ctx, domainName)

	result.ScanDuration = int(time.Since(start).Milliseconds())

	logger.Info().
		Str("domain", domainName).
		Int("records", len(result.Records)).
		Int("subdomains", len(result.Subdomains)).
		Int("duration_ms", result.ScanDuration).
		Msg("DNS scan completed")

	return result, nil
}

func (s *DNSScanner) getDNSRecords(ctx context.Context, domainName string) []domain.DNSRecord {
	var records []domain.DNSRecord

	// A and AAAA records
	ips, err := s.resolver.LookupIPAddr(ctx, domainName)
	if err == nil {
		for _, ip := range ips {
			rtype := "A"
			if ip.IP.To4() == nil {
				rtype = "AAAA"
			}
			records = append(records, domain.DNSRecord{
				Type:  rtype,
				Name:  domainName,
				Value: ip.String(),
			})
		}
	}

	// CNAME
	cname, err := s.resolver.LookupCNAME(ctx, domainName)
	if err == nil && cname != domainName+"." {
		records = append(records, domain.DNSRecord{
			Type:  "CNAME",
			Name:  domainName,
			Value: strings.TrimSuffix(cname, "."),
		})
	}

	// TXT records
	txts, err := s.resolver.LookupTXT(ctx, domainName)
	if err == nil {
		for _, txt := range txts {
			records = append(records, domain.DNSRecord{
				Type:  "TXT",
				Name:  domainName,
				Value: txt,
			})
		}
	}

	return records
}

func (s *DNSScanner) getNameservers(ctx context.Context, domainName string) []string {
	nss, err := s.resolver.LookupNS(ctx, domainName)
	if err != nil {
		return nil
	}
	var result []string
	for _, ns := range nss {
		result = append(result, strings.TrimSuffix(ns.Host, "."))
	}
	return result
}

func (s *DNSScanner) getMXRecords(ctx context.Context, domainName string) []string {
	mxs, err := s.resolver.LookupMX(ctx, domainName)
	if err != nil {
		return nil
	}
	var result []string
	for _, mx := range mxs {
		result = append(result, fmt.Sprintf("%s (priority: %d)", strings.TrimSuffix(mx.Host, "."), mx.Pref))
	}
	return result
}

func (s *DNSScanner) getSPFRecord(ctx context.Context, domainName string) string {
	txts, err := s.resolver.LookupTXT(ctx, domainName)
	if err != nil {
		return ""
	}
	for _, txt := range txts {
		if strings.HasPrefix(strings.ToLower(txt), "v=spf1") {
			return txt
		}
	}
	return ""
}

func (s *DNSScanner) getDMARCRecord(ctx context.Context, domainName string) string {
	txts, err := s.resolver.LookupTXT(ctx, "_dmarc."+domainName)
	if err != nil {
		return ""
	}
	for _, txt := range txts {
		if strings.HasPrefix(strings.ToLower(txt), "v=dmarc1") {
			return txt
		}
	}
	return ""
}

func (s *DNSScanner) enumerateSubdomains(ctx context.Context, baseDomain string) []domain.SubdomainResult {
	// Common government website subdomains
	commonSubs := []string{
		"www", "mail", "webmail", "remote", "blog", "dev", "staging",
		"api", "app", "admin", "portal", "vpn", "ftp", "ssh",
		"test", "demo", "backup", "old", "new", "beta",
		"cdn", "static", "media", "assets", "img", "images",
		"db", "database", "sql", "mysql", "phpmyadmin",
		"cpanel", "whm", "plesk", "webmin",
		"ns1", "ns2", "dns", "dns1", "dns2",
		"mx", "smtp", "pop", "imap", "exchange",
		"owa", "autodiscover", "lyncdiscover",
		"sip", "sipdir", "lync",
		"login", "sso", "auth", "cas", "idp",
		"jira", "confluence", "gitlab", "git", "svn",
		"jenkins", "ci", "build", "deploy",
		"monitor", "monitoring", "nagios", "zabbix", "grafana",
		"elk", "kibana", "elasticsearch", "logstash",
		"docs", "wiki", "help", "support", "ticket",
		"forum", "community", "chat",
		"store", "shop", "ecommerce",
		"crm", "erp", "hr", "finance",
		"intranet", "internal", "private",
		"cloud", "storage", "files", "share",
		"proxy", "gateway", "firewall",
		"waf", "ids", "ips",
		"m", "mobile", "wap",
		"v1", "v2", "v3",
		"web", "web1", "web2", "web3",
		"server", "server1", "server2",
		"node1", "node2", "worker",
		"cache", "redis", "memcached",
		"queue", "mq", "rabbitmq",
		"report", "reports", "analytics",
		"dashboard", "panel",
		"e-office", "eoffice", "sikd", "simda", "sipd",
		"lpse", "sirup", "span", "sakti",
		"ppdb", "siakad", "simak",
	}

	var results []domain.SubdomainResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use semaphore for concurrency control
	sem := make(chan struct{}, 20)

	titleRe := regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)

	for _, sub := range commonSubs {
		select {
		case <-ctx.Done():
			return results
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(subdomain string) {
			defer wg.Done()
			defer func() { <-sem }()

			fqdn := subdomain + "." + baseDomain

			// DNS lookup
			ips, err := s.resolver.LookupIPAddr(ctx, fqdn)
			if err != nil || len(ips) == 0 {
				return
			}

			result := domain.SubdomainResult{
				Subdomain: fqdn,
				IP:        ips[0].String(),
				FoundAt:   time.Now(),
				Source:     "dns",
			}

			// Try HTTP request for title and status
			resp, err := s.httpClient.Get("https://" + fqdn)
			if err != nil {
				resp, err = s.httpClient.Get("http://" + fqdn)
			}
			if err == nil {
				result.StatusCode = resp.StatusCode
				// Read limited body for title
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 50000))
				resp.Body.Close()
				if matches := titleRe.FindSubmatch(body); len(matches) > 1 {
					result.Title = strings.TrimSpace(string(matches[1]))
					if len(result.Title) > 100 {
						result.Title = result.Title[:100]
					}
				}
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(sub)
	}

	wg.Wait()
	return results
}

// ScanAllWebsites scans DNS for all active websites, deduplicating by root domain.
// Uses up to 5 concurrent goroutines to avoid blocking on slow domains.
func (s *DNSScanner) ScanAllWebsites(ctx context.Context) error {
	websites, _, err := s.websiteRepo.GetAll(ctx, domain.WebsiteFilter{Limit: 1000})
	if err != nil {
		return err
	}

	// Deduplicate by root domain before dispatching goroutines
	type workItem struct {
		website    domain.Website
		rootDomain string
	}
	seen := make(map[string]bool)
	var items []workItem

	for _, w := range websites {
		parsedURL, err := url.Parse(w.URL)
		if err != nil {
			logger.Warn().Err(err).Str("url", w.URL).Msg("DNS scan: skipping website with invalid URL")
			continue
		}
		rootDomain := parsedURL.Hostname()
		// Extract root domain (remove subdomain)
		parts := strings.Split(rootDomain, ".")
		if len(parts) > 2 {
			rootDomain = strings.Join(parts[len(parts)-2:], ".")
		}
		if seen[rootDomain] {
			continue
		}
		seen[rootDomain] = true
		items = append(items, workItem{website: w, rootDomain: rootDomain})
	}

	total := len(items)
	logger.Info().Int("total_domains", total).Msg("DNS scan: starting concurrent scan")

	// Semaphore: max 5 concurrent DNS scans
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for idx, item := range items {
		if ctx.Err() != nil {
			logger.Warn().Msg("DNS scan: context cancelled, stopping early")
			break
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(i int, it workItem) {
			defer wg.Done()
			defer func() { <-sem }()

			logger.Info().
				Int("index", i+1).
				Int("total", total).
				Str("domain", it.rootDomain).
				Msg("DNS scan: scanning domain")

			// Per-website timeout: 60s to allow subdomain enumeration to complete
			scanCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			wCopy := it.website
			result, scanErr := s.ScanDNS(scanCtx, &wCopy)

			if scanErr != nil {
				logger.Error().
					Err(scanErr).
					Str("domain", it.rootDomain).
					Int("index", i+1).
					Int("total", total).
					Msg("DNS scan: failed for domain, continuing with next")
				return
			}

			// Save result to database
			if s.dnsRepo != nil {
				record := &domain.DNSScanRecord{
					WebsiteID:      result.WebsiteID,
					DomainName:     result.Domain,
					HasSPF:         result.SPFRecord != "",
					HasDMARC:       result.DMARCRecord != "",
					SPFRecord:      result.SPFRecord,
					DMARCRecord:    result.DMARCRecord,
					Nameservers:    result.Nameservers,
					MXRecords:      result.MXRecords,
					DNSRecords:     result.Records,
					Subdomains:     result.Subdomains,
					SubdomainCount: len(result.Subdomains),
					ScanDurationMs: result.ScanDuration,
				}
				if saveErr := s.dnsRepo.SaveDNSScan(ctx, record); saveErr != nil {
					logger.Error().Err(saveErr).Str("domain", it.rootDomain).Msg("DNS scan: failed to save result")
				} else {
					logger.Info().
						Str("domain", it.rootDomain).
						Int("records", len(result.Records)).
						Int("subdomains", len(result.Subdomains)).
						Int("index", i+1).
						Int("total", total).
						Msg("DNS scan: saved result")
				}
			}
		}(idx, item)
	}

	wg.Wait()
	logger.Info().Int("total_domains", total).Msg("DNS scan: all domains completed")
	return nil
}
