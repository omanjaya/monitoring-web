package notifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type EmailNotifier struct {
	cfg       *config.Config
	alertRepo *mysql.AlertRepository
}

func NewEmailNotifier(cfg *config.Config, alertRepo *mysql.AlertRepository) *EmailNotifier {
	return &EmailNotifier{
		cfg:       cfg,
		alertRepo: alertRepo,
	}
}

// SendAlert sends an alert notification via Email
func (e *EmailNotifier) SendAlert(ctx context.Context, alert *domain.Alert, website *domain.Website) error {
	if !e.cfg.Email.Enabled {
		logger.Debug().Msg("Email notifications disabled")
		return nil
	}

	subject := e.formatSubject(alert, website)
	body := e.formatAlertBody(alert, website)

	// Send to all configured recipients
	for _, recipient := range e.cfg.Email.Recipients {
		// Create notification record
		notification := &domain.Notification{
			AlertID:   alert.ID,
			Channel:   "email",
			Recipient: recipient,
			Status:    "pending",
		}

		notifID, err := e.alertRepo.CreateNotification(ctx, notification)
		if err != nil {
			logger.Error().Err(err).Str("recipient", recipient).Msg("Failed to create email notification record")
			continue
		}

		// Send email with retry
		err = retryWithBackoff("email", 3, func() error {
			return e.sendEmail(recipient, subject, body)
		})
		if err != nil {
			logger.Error().Err(err).Str("recipient", recipient).Msg("Failed to send email after retries")
			e.alertRepo.UpdateNotificationStatus(ctx, notifID, "failed", err.Error())
			continue
		}

		e.alertRepo.UpdateNotificationStatus(ctx, notifID, "sent", "")
		logger.Info().Str("recipient", recipient).Int64("alert_id", alert.ID).Msg("Email notification sent")
	}

	return nil
}

func (e *EmailNotifier) sendEmail(to, subject, body string) error {
	cfg := e.cfg.Email

	// Build headers
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", cfg.FromName, cfg.From)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	// Build message
	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	// Setup authentication
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	if cfg.UseTLS {
		return e.sendEmailTLS(addr, auth, cfg.From, to, message.String())
	}

	return smtp.SendMail(addr, auth, cfg.From, []string{to}, []byte(message.String()))
}

func (e *EmailNotifier) sendEmailTLS(addr string, auth smtp.Auth, from, to, message string) error {
	cfg := e.cfg.Email

	// TLS config
	tlsconfig := &tls.Config{
		ServerName: cfg.SMTPHost,
	}

	// Connect to server
	conn, err := tls.Dial("tcp", addr, tlsconfig)
	if err != nil {
		// Try STARTTLS instead
		return e.sendEmailSTARTTLS(addr, auth, from, to, message)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Close()

	// Auth
	if err = client.Auth(auth); err != nil {
		return err
	}

	// Set sender and recipient
	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}

	// Send body
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

func (e *EmailNotifier) sendEmailSTARTTLS(addr string, auth smtp.Auth, from, to, message string) error {
	cfg := e.cfg.Email

	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	// STARTTLS
	tlsconfig := &tls.Config{
		ServerName: cfg.SMTPHost,
	}
	if err = client.StartTLS(tlsconfig); err != nil {
		return err
	}

	// Auth
	if err = client.Auth(auth); err != nil {
		return err
	}

	// Set sender and recipient
	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}

	// Send body
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

func (e *EmailNotifier) formatSubject(alert *domain.Alert, website *domain.Website) string {
	var prefix string
	switch alert.Severity {
	case domain.SeverityCritical:
		prefix = "[CRITICAL]"
	case domain.SeverityWarning:
		prefix = "[WARNING]"
	case domain.SeverityInfo:
		prefix = "[INFO]"
	}

	websiteName := ""
	if website != nil {
		websiteName = website.Name
	}

	return fmt.Sprintf("%s %s - %s", prefix, alert.Title, websiteName)
}

func (e *EmailNotifier) formatAlertBody(alert *domain.Alert, website *domain.Website) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .header.critical { background-color: #dc3545; color: white; }
        .header.warning { background-color: #ffc107; color: #333; }
        .header.info { background-color: #17a2b8; color: white; }
        .content { background-color: #f8f9fa; padding: 20px; border: 1px solid #dee2e6; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: bold; color: #495057; }
        .footer { padding: 15px; text-align: center; font-size: 12px; color: #6c757d; }
        .badge { display: inline-block; padding: 5px 10px; border-radius: 3px; font-weight: bold; }
        .badge-down { background-color: #dc3545; color: white; }
        .badge-up { background-color: #28a745; color: white; }
        .badge-warning { background-color: #ffc107; color: #333; }
    </style>
</head>
<body>
    <div class="container">
`)

	// Header based on severity
	headerClass := "info"
	headerTitle := "INFO"
	switch alert.Severity {
	case domain.SeverityCritical:
		headerClass = "critical"
		headerTitle = "CRITICAL ALERT"
	case domain.SeverityWarning:
		headerClass = "warning"
		headerTitle = "WARNING"
	}

	sb.WriteString(fmt.Sprintf(`        <div class="header %s">
            <h1>%s</h1>
        </div>
        <div class="content">
`, headerClass, headerTitle))

	// Alert type badge
	alertBadge := ""
	switch alert.Type {
	case domain.AlertTypeDown:
		alertBadge = `<span class="badge badge-down">WEBSITE DOWN</span>`
	case domain.AlertTypeUp:
		alertBadge = `<span class="badge badge-up">WEBSITE UP</span>`
	case domain.AlertTypeSlowResponse:
		alertBadge = `<span class="badge badge-warning">SLOW RESPONSE</span>`
	case domain.AlertTypeSSLExpired:
		alertBadge = `<span class="badge badge-down">SSL EXPIRED</span>`
	case domain.AlertTypeSSLExpiring:
		alertBadge = `<span class="badge badge-warning">SSL EXPIRING</span>`
	case domain.AlertTypeJudolDetected:
		alertBadge = `<span class="badge badge-down">JUDOL DETECTED</span>`
	case domain.AlertTypeDefacement:
		alertBadge = `<span class="badge badge-down">DEFACEMENT</span>`
	}

	sb.WriteString(fmt.Sprintf(`            <div class="detail-row">%s</div>
`, alertBadge))

	// Website info
	if website != nil {
		sb.WriteString(fmt.Sprintf(`            <div class="detail-row">
                <span class="label">Website:</span> %s
            </div>
            <div class="detail-row">
                <span class="label">URL:</span> <a href="%s">%s</a>
            </div>
`, website.Name, website.URL, website.URL))

		if website.OPD != nil {
			sb.WriteString(fmt.Sprintf(`            <div class="detail-row">
                <span class="label">OPD:</span> %s
            </div>
`, website.OPD.Name))
		}
	}

	// Alert details
	sb.WriteString(fmt.Sprintf(`            <div class="detail-row">
                <span class="label">Title:</span> %s
            </div>
            <div class="detail-row">
                <span class="label">Message:</span> %s
            </div>
            <div class="detail-row">
                <span class="label">Time:</span> %s
            </div>
`, alert.Title, alert.Message, alert.CreatedAt.Format("02 Jan 2006 15:04:05 WIB")))

	sb.WriteString(`        </div>
        <div class="footer">
            Monitoring Website - Diskominfos Provinsi Bali<br>
            Email ini dikirim secara otomatis, mohon tidak membalas email ini.
        </div>
    </div>
</body>
</html>`)

	return sb.String()
}

// SendTestMessage sends a test email to verify configuration
func (e *EmailNotifier) SendTestMessage(ctx context.Context) error {
	if !e.cfg.Email.Enabled {
		return fmt.Errorf("email notifications are disabled")
	}

	subject := "[TEST] Email Notification - Monitoring Website"
	body := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #28a745; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f8f9fa; padding: 20px; border: 1px solid #dee2e6; }
        .footer { padding: 15px; text-align: center; font-size: 12px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>TEST MESSAGE</h1>
        </div>
        <div class="content">
            <p>Email notification berhasil dikonfigurasi!</p>
            <p>Anda akan menerima notifikasi alert melalui email ini.</p>
        </div>
        <div class="footer">
            Monitoring Website - Diskominfos Provinsi Bali
        </div>
    </div>
</body>
</html>`

	for _, recipient := range e.cfg.Email.Recipients {
		if err := e.sendEmail(recipient, subject, body); err != nil {
			return fmt.Errorf("failed to send to %s: %w", recipient, err)
		}
	}

	return nil
}

// SendDailySummary sends daily monitoring summary via email
func (e *EmailNotifier) SendDailySummary(ctx context.Context, summary *DailySummary) error {
	if !e.cfg.Email.Enabled {
		return nil
	}

	subject := fmt.Sprintf("[DAILY REPORT] Monitoring Website - %s", time.Now().Format("02 Jan 2006"))
	body := e.formatDailySummaryBody(summary)

	for _, recipient := range e.cfg.Email.Recipients {
		if err := e.sendEmail(recipient, subject, body); err != nil {
			logger.Error().Err(err).Str("recipient", recipient).Msg("Failed to send daily summary email")
		}
	}

	return nil
}

func (e *EmailNotifier) formatDailySummaryBody(summary *DailySummary) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #007bff; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f8f9fa; padding: 20px; border: 1px solid #dee2e6; }
        .stat-box { display: inline-block; width: 45%; margin: 5px; padding: 15px; background: white; border-radius: 5px; text-align: center; }
        .stat-value { font-size: 24px; font-weight: bold; }
        .stat-label { font-size: 12px; color: #6c757d; }
        .stat-up { color: #28a745; }
        .stat-down { color: #dc3545; }
        .stat-warning { color: #ffc107; }
        .footer { padding: 15px; text-align: center; font-size: 12px; color: #6c757d; }
        table { width: 100%; border-collapse: collapse; margin-top: 15px; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #dee2e6; }
        th { background-color: #e9ecef; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>LAPORAN HARIAN MONITORING</h1>
            <p>`)
	sb.WriteString(time.Now().Format("02 January 2006"))
	sb.WriteString(`</p>
        </div>
        <div class="content">
            <h3>Status Website</h3>
            <div style="text-align: center;">
                <div class="stat-box">
                    <div class="stat-value stat-up">`)
	sb.WriteString(fmt.Sprintf("%d", summary.WebsitesUp))
	sb.WriteString(`</div>
                    <div class="stat-label">UP</div>
                </div>
                <div class="stat-box">
                    <div class="stat-value stat-down">`)
	sb.WriteString(fmt.Sprintf("%d", summary.WebsitesDown))
	sb.WriteString(`</div>
                    <div class="stat-label">DOWN</div>
                </div>
                <div class="stat-box">
                    <div class="stat-value stat-warning">`)
	sb.WriteString(fmt.Sprintf("%d", summary.WebsitesDegraded))
	sb.WriteString(`</div>
                    <div class="stat-label">DEGRADED</div>
                </div>
                <div class="stat-box">
                    <div class="stat-value">`)
	sb.WriteString(fmt.Sprintf("%d", summary.TotalWebsites))
	sb.WriteString(`</div>
                    <div class="stat-label">TOTAL</div>
                </div>
            </div>

            <h3>Alert Summary</h3>
            <table>
                <tr>
                    <th>Severity</th>
                    <th>Count</th>
                </tr>
                <tr>
                    <td>Critical</td>
                    <td class="stat-down">`)
	sb.WriteString(fmt.Sprintf("%d", summary.CriticalAlerts))
	sb.WriteString(`</td>
                </tr>
                <tr>
                    <td>Warning</td>
                    <td class="stat-warning">`)
	sb.WriteString(fmt.Sprintf("%d", summary.WarningAlerts))
	sb.WriteString(`</td>
                </tr>
                <tr>
                    <td>Info</td>
                    <td>`)
	sb.WriteString(fmt.Sprintf("%d", summary.InfoAlerts))
	sb.WriteString(`</td>
                </tr>
            </table>

            <h3>Performance</h3>
            <table>
                <tr>
                    <td>Average Response Time</td>
                    <td>`)
	sb.WriteString(fmt.Sprintf("%d ms", summary.AvgResponseTime))
	sb.WriteString(`</td>
                </tr>
                <tr>
                    <td>Uptime</td>
                    <td>`)
	sb.WriteString(fmt.Sprintf("%.2f%%", summary.UptimePercentage))
	sb.WriteString(`</td>
                </tr>
            </table>
`)

	if summary.JudolDetected > 0 {
		sb.WriteString(fmt.Sprintf(`
            <div style="background-color: #dc3545; color: white; padding: 15px; margin-top: 15px; border-radius: 5px;">
                <strong>PERINGATAN:</strong> Terdeteksi %d website dengan konten judi online!
            </div>
`, summary.JudolDetected))
	}

	sb.WriteString(`
        </div>
        <div class="footer">
            Monitoring Website - Diskominfos Provinsi Bali<br>
            Email ini dikirim secara otomatis setiap hari.
        </div>
    </div>
</body>
</html>`)

	return sb.String()
}

// SendAlertResolved sends notification when an alert is resolved
func (e *EmailNotifier) SendAlertResolved(ctx context.Context, alert *domain.Alert, website *domain.Website, note string) error {
	if !e.cfg.Email.Enabled {
		return nil
	}

	websiteName := ""
	if website != nil {
		websiteName = website.Name
	}

	subject := fmt.Sprintf("[RESOLVED] %s - %s", alert.Title, websiteName)
	body := e.formatResolvedBody(alert, website, note)

	for _, recipient := range e.cfg.Email.Recipients {
		if err := e.sendEmail(recipient, subject, body); err != nil {
			logger.Error().Err(err).Str("recipient", recipient).Msg("Failed to send resolved email")
		}
	}

	return nil
}

func (e *EmailNotifier) formatResolvedBody(alert *domain.Alert, website *domain.Website, note string) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #28a745; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f8f9fa; padding: 20px; border: 1px solid #dee2e6; }
        .detail-row { margin: 10px 0; }
        .label { font-weight: bold; color: #495057; }
        .footer { padding: 15px; text-align: center; font-size: 12px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>ALERT RESOLVED</h1>
        </div>
        <div class="content">
`)

	// Resolution message based on alert type
	resolutionMsg := "Issue sudah diselesaikan"
	switch alert.Type {
	case domain.AlertTypeDown:
		resolutionMsg = "Website kembali online"
	case domain.AlertTypeSSLExpired, domain.AlertTypeSSLExpiring:
		resolutionMsg = "SSL certificate sudah diperbarui"
	case domain.AlertTypeJudolDetected:
		resolutionMsg = "Konten sudah dibersihkan"
	}

	sb.WriteString(fmt.Sprintf(`            <p style="font-size: 18px; color: #28a745;">%s</p>
`, resolutionMsg))

	// Website info
	if website != nil {
		sb.WriteString(fmt.Sprintf(`            <div class="detail-row">
                <span class="label">Website:</span> %s
            </div>
            <div class="detail-row">
                <span class="label">URL:</span> <a href="%s">%s</a>
            </div>
`, website.Name, website.URL, website.URL))
	}

	// Note
	if note != "" {
		sb.WriteString(fmt.Sprintf(`            <div class="detail-row">
                <span class="label">Note:</span> %s
            </div>
`, note))
	}

	sb.WriteString(fmt.Sprintf(`            <div class="detail-row">
                <span class="label">Resolved at:</span> %s
            </div>
`, time.Now().Format("02 Jan 2006 15:04:05 WIB")))

	sb.WriteString(`        </div>
        <div class="footer">
            Monitoring Website - Diskominfos Provinsi Bali
        </div>
    </div>
</body>
</html>`)

	return sb.String()
}

// SendRawEmail sends a raw email to a specific recipient (used for escalations)
func (e *EmailNotifier) SendRawEmail(ctx context.Context, recipient, subject, body string) error {
	if !e.cfg.Email.Enabled {
		return nil
	}
	return e.sendEmail(recipient, subject, body)
}
