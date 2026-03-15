-- Add AI verification tracking columns
ALTER TABLE dork_detections ADD COLUMN ai_verified BOOLEAN NOT NULL DEFAULT FALSE AFTER confidence;

ALTER TABLE dork_scan_results ADD COLUMN ai_filtered_count INT NOT NULL DEFAULT 0 AFTER low_count;
