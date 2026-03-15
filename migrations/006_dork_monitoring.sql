-- Dork Monitoring Migration
-- This migration creates tables for Google Dork-style monitoring

-- Dork patterns table (custom patterns beyond defaults)
CREATE TABLE IF NOT EXISTS dork_patterns (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    category ENUM('gambling', 'defacement', 'malware', 'phishing', 'seo_spam', 'webshell', 'backdoor', 'injection', 'custom') NOT NULL,
    pattern TEXT NOT NULL,
    pattern_type ENUM('keyword', 'regex', 'xpath', 'css_selector') NOT NULL DEFAULT 'keyword',
    severity ENUM('critical', 'high', 'medium', 'low', 'info') NOT NULL DEFAULT 'medium',
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_category (category),
    INDEX idx_severity (severity),
    INDEX idx_is_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dork scan results table
CREATE TABLE IF NOT EXISTS dork_scan_results (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    website_id BIGINT UNSIGNED NOT NULL,
    scan_type ENUM('full', 'quick', 'targeted') NOT NULL DEFAULT 'quick',
    status ENUM('pending', 'running', 'completed', 'failed') NOT NULL DEFAULT 'pending',
    total_pages_scanned INT NOT NULL DEFAULT 0,
    total_detections INT NOT NULL DEFAULT 0,
    critical_count INT NOT NULL DEFAULT 0,
    high_count INT NOT NULL DEFAULT 0,
    medium_count INT NOT NULL DEFAULT 0,
    low_count INT NOT NULL DEFAULT 0,
    categories_scanned JSON,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,
    INDEX idx_website_id (website_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dork detections table (individual findings)
CREATE TABLE IF NOT EXISTS dork_detections (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    scan_result_id BIGINT UNSIGNED NOT NULL,
    website_id BIGINT UNSIGNED NOT NULL,
    pattern_id BIGINT UNSIGNED NULL,
    pattern_name VARCHAR(255) NOT NULL,
    category ENUM('gambling', 'defacement', 'malware', 'phishing', 'seo_spam', 'webshell', 'backdoor', 'injection', 'custom') NOT NULL,
    severity ENUM('critical', 'high', 'medium', 'low', 'info') NOT NULL,
    url TEXT NOT NULL,
    matched_content TEXT,
    context TEXT,
    confidence DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    is_false_positive BOOLEAN NOT NULL DEFAULT FALSE,
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMP NULL,
    resolved_by VARCHAR(255),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (scan_result_id) REFERENCES dork_scan_results(id) ON DELETE CASCADE,
    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,
    FOREIGN KEY (pattern_id) REFERENCES dork_patterns(id) ON DELETE SET NULL,
    INDEX idx_scan_result_id (scan_result_id),
    INDEX idx_website_id (website_id),
    INDEX idx_category (category),
    INDEX idx_severity (severity),
    INDEX idx_is_resolved (is_resolved),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Website dork settings (per-website configuration)
CREATE TABLE IF NOT EXISTS website_dork_settings (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    website_id BIGINT UNSIGNED NOT NULL UNIQUE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    scan_frequency ENUM('hourly', 'daily', 'weekly', 'manual') NOT NULL DEFAULT 'daily',
    scan_depth INT NOT NULL DEFAULT 3,
    max_pages INT NOT NULL DEFAULT 50,
    categories_enabled JSON,
    excluded_paths JSON,
    last_scan_at TIMESTAMP NULL,
    next_scan_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,
    INDEX idx_is_enabled (is_enabled),
    INDEX idx_next_scan_at (next_scan_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Insert default dork patterns
INSERT INTO dork_patterns (name, category, pattern, pattern_type, severity, description, is_default) VALUES
-- =============================================
-- GAMBLING (JUDOL) PATTERNS - Indonesian Specific
-- Source: Research paper arxiv.org/html/2508.19368v1, GbDetector GitHub, BrigsLabs/judol
-- =============================================

-- Core Detection Keywords (from arxiv research - lowest false positive rate)
('Core Gambling Keywords', 'gambling', 'togel|toto|judi|slot|gacor|bandar|maxwin|zeus', 'regex', 'critical', 'Kata kunci inti deteksi judi (penelitian arxiv)', TRUE),

-- Slot Keywords
('Slot Gacor Basic', 'gambling', 'slot gacor|slot online|slot88|slot777|slot138|slot212|slot4d|slot5000|slot dana|slot pulsa', 'regex', 'critical', 'Deteksi kata kunci slot online dasar', TRUE),
('Slot Provider Pragmatic', 'gambling', 'pragmatic play|pragmatic88|pragmaticplay|pp slot|pragmatic slot|pragmatic demo', 'regex', 'critical', 'Deteksi provider Pragmatic Play', TRUE),
('Slot Provider PGSoft', 'gambling', 'pg soft|pgsoft|pg slot|mahjong ways|mahjong wins|fortune mouse|lucky neko|fortune rabbit', 'regex', 'critical', 'Deteksi provider PG Soft', TRUE),
('Slot Provider Others', 'gambling', 'joker123|joker388|joker gaming|habanero|spadegaming|microgaming|playtech|netent|isoftbet|nolimit city|cq9|jili|fachai', 'regex', 'critical', 'Deteksi provider slot lainnya', TRUE),
('Slot Thailand', 'gambling', 'slot thailand|slot server thailand|slot luar negeri|akun pro thailand|slot server luar|server thailand', 'regex', 'critical', 'Deteksi slot server Thailand', TRUE),
('Slot Maxwin', 'gambling', 'maxwin|scatter hitam|scatter petir|anti rungkad|anti lag|pasti jp|gampang menang|auto sultan|rtp live|rtp gacor|rtp slot', 'regex', 'critical', 'Deteksi kata kunci slot maxwin', TRUE),

-- Popular Slot Games
('Slot Games Olympus', 'gambling', 'gates of olympus|olympus slot|kakek zeus|dewa zeus|zeus slot|gates of gatot|gatot kaca', 'regex', 'critical', 'Deteksi game slot Olympus/Zeus', TRUE),
('Slot Games Bonanza', 'gambling', 'sweet bonanza|bonanza slot|sugar rush|candy burst|fruit party|candy village', 'regex', 'critical', 'Deteksi game slot Bonanza series', TRUE),
('Slot Games Princess', 'gambling', 'starlight princess|princess slot|koi princess|moon princess|rise of princess', 'regex', 'critical', 'Deteksi game slot Princess series', TRUE),
('Slot Games Mahjong', 'gambling', 'mahjong ways|mahjong ways 2|mahjong wins|mahjong panda|mahjong slot', 'regex', 'critical', 'Deteksi game slot Mahjong series', TRUE),
('Slot Games Others', 'gambling', 'wild west gold|aztec gems|fortune tiger|lucky neko|fortune ox|dragon tiger luck|book of ra|book of dead|wolf gold', 'regex', 'critical', 'Deteksi game slot populer lainnya', TRUE),

-- Situs Judi Populer (with leetspeak) - Comprehensive
('Raja Series', 'gambling', 'rajabola|rajabol4|rajab0la|raj4bola|rajacuan|rajaslot|rajatogel|rajat0gel|rajapoker|rajabet|raja388|raja189|raja69|raja99|rajawin|rajajp|rajahoki', 'regex', 'critical', 'Deteksi situs raja series', TRUE),
('Dewa Series', 'gambling', 'dewabet|dewa288|dewacasino|dewapoker|dewatogel|dewajudi|dewa303|dewa234|dewacash|dewaslot|dewawin|dewahoki|dewa88|dewa99|dewacuan', 'regex', 'critical', 'Deteksi situs dewa series', TRUE),
('Boss Series', 'gambling', 'bos88|bosku|bosswin|bosslot|bosjp|bostogel|bosqq|bosbet|boss88|boscuan|bosshoki|bos303|bos99|bosjudi|bosspoker', 'regex', 'critical', 'Deteksi situs boss series', TRUE),
('MPO Series', 'gambling', 'mpo88|mpo100|mpo188|mpo189|mpo500|mpo777|mpo868|mposlot|mpoplay|mpocash|mpo388|mpo555|mpo99|mpo303|mpobos|mpohoki|mpowin', 'regex', 'critical', 'Deteksi situs MPO series', TRUE),
('Gacor Series', 'gambling', 'gacor88|gacor77|gacor123|gacorjp|gacor4d|gacor188|gacorbet|gacorwin|slotgacor|gacor303|gacormania|gacortoto|gacorcuan|gacoremas', 'regex', 'critical', 'Deteksi situs gacor series', TRUE),
('Hoki Series', 'gambling', 'hoki88|hoki99|hoki777|hokibet|hokijudi|hokislot|lucky88|lucky77|luckybet|lucky303|hokicuan|hokijp|hokiwin|hokinaga|hokiemas', 'regex', 'critical', 'Deteksi situs hoki/lucky series', TRUE),
('Win Series', 'gambling', 'win88|win99|win138|win303|winbet|winjudi|menang88|menang123|menangbet|menangslot|wintoto|wincuan|winhoki|winjp|winslot', 'regex', 'critical', 'Deteksi situs win/menang series', TRUE),
('JP Series', 'gambling', 'jp88|jp99|jp138|jp303|jpslot|jptogel|jp4d|jackpot88|jackpot99|jackpotslot|jphoki|jpwin|jpcuan|jpemas|jppaus', 'regex', 'critical', 'Deteksi situs jackpot series', TRUE),
('Cuan Series', 'gambling', 'cuan88|cuan99|cuan123|cuanbet|cuanslot|cuanjp|cuantogel|cuan4d|cuan303|cuanwin|cuanhoki|cuanemas|datacuan|cuanbesar', 'regex', 'critical', 'Deteksi situs cuan series', TRUE),
('88 Series Extended', 'gambling', 'indo88|asia88|emas88|uang88|rupiah88|duit88|bola88|casino88|poker88|togel88|toto88|naga88|dragon88|sultan88|mega88|super88|power88|raja88|dewa88|hot88|vip88|royal88', 'regex', 'critical', 'Deteksi situs dengan angka 88', TRUE),
('303 Series Extended', 'gambling', 'asia303|indo303|euro303|vegas303|naga303|dragon303|slot303|togel303|casino303|sultan303|mega303|super303|royal303|vip303|poker303', 'regex', 'critical', 'Deteksi situs dengan angka 303', TRUE),
('4D Series', 'gambling', 'slot4d|togel4d|toto4d|judi4d|situs4d|link4d|hoki4d|jp4d|cuan4d|win4d|play4d|vip4d|mega4d', 'regex', 'critical', 'Deteksi situs dengan 4D', TRUE),
('Naga Series', 'gambling', 'naga88|naga303|nagaslot|nagatogel|nagapoker|nagabet|nagacuan|nagahoki|nagawin|nagajp|nagaemas', 'regex', 'critical', 'Deteksi situs naga series', TRUE),
('Dragon Series', 'gambling', 'dragon88|dragon303|dragonslot|dragontogel|dragonbet|dragonwin|dragonhoki|dragonjp', 'regex', 'critical', 'Deteksi situs dragon series', TRUE),
('Sultan Series', 'gambling', 'sultan88|sultan303|sultanslot|sultantogel|sultanbet|sultanwin|sultanhoki|sultanjp|sultanplay', 'regex', 'critical', 'Deteksi situs sultan series', TRUE),

-- Leetspeak Comprehensive Patterns
('Leetspeak Pattern Basic', 'gambling', '(r[a4]j[a4]|d[e3]w[a4]|b[o0]s|h[o0]k[i1]|w[i1]n|g[a4]c[o0]r|cu[a4]n|m[a4]xw[i1]n|j[a4]ckp[o0]t)(b[o0]l[a4]|sl[o0]t|t[o0]g[e3]l|c[a4]s[i1]n[o0]|p[o0]k[e3]r|b[e3]t|jud[i1]|qq)?\\d*', 'regex', 'critical', 'Deteksi nama judi dengan leetspeak', TRUE),
('Leetspeak Slot', 'gambling', 'sl[o0]t\\s*g[a4]c[o0]r|sl[o0]t\\s*[o0]nl[i1]n[e3]|sl[o0]t\\s*m[a4]xw[i1]n|sl[o0]t88|sl[o0]t777', 'regex', 'critical', 'Deteksi slot dengan leetspeak', TRUE),
('Leetspeak Togel', 'gambling', 't[o0]g[e3]l|t[o0]t[o0]\\s*(g[e3]l[a4]p)?|b[a4]nd[a4]r\\s*t[o0]g[e3]l', 'regex', 'critical', 'Deteksi togel dengan leetspeak', TRUE),
('Leetspeak Judi', 'gambling', 'jud[i1]\\s*[o0]nl[i1]n[e3]|jud[o0]l|j[a4]d[o0]l|s[i1]tus\\s*jud[i1]', 'regex', 'critical', 'Deteksi judi dengan leetspeak', TRUE),

-- Character Insertion Evasion (from GbDetector)
('Evasion Dots', 'gambling', 's\\.?l\\.?o\\.?t|t\\.?o\\.?g\\.?e\\.?l|j\\.?u\\.?d\\.?i|g\\.?a\\.?c\\.?o\\.?r|c\\.?a\\.?s\\.?i\\.?n\\.?o', 'regex', 'high', 'Deteksi evasion dengan titik', TRUE),
('Evasion Spaces', 'gambling', 's\\s*l\\s*o\\s*t|t\\s*o\\s*g\\s*e\\s*l|j\\s*u\\s*d\\s*i|g\\s*a\\s*c\\s*o\\s*r', 'regex', 'high', 'Deteksi evasion dengan spasi', TRUE),
('Evasion Stars', 'gambling', 's\\*l\\*o\\*t|t\\*o\\*g\\*e\\*l|j\\*u\\*d\\*i|g\\*a\\*c\\*o\\*r', 'regex', 'high', 'Deteksi evasion dengan asterisk', TRUE),
('Evasion Dashes', 'gambling', 's-l-o-t|t-o-g-e-l|j-u-d-i|g-a-c-o-r|c-a-s-i-n-o', 'regex', 'high', 'Deteksi evasion dengan dash', TRUE),

-- Togel Keywords Comprehensive
('Togel Online', 'gambling', 'togel online|togel singapore|togel hongkong|togel sydney|togel sgp|togel hk|togel sdy|togel macau|togel cambodia|togel taipei|togel japan', 'regex', 'critical', 'Deteksi kata kunci togel online', TRUE),
('Togel Pools', 'gambling', 'singapore pools|hongkong pools|sydney pools|sgp pools|hk pools|sdy pools|toto macau pools|pools resmi', 'regex', 'critical', 'Deteksi togel pools', TRUE),
('Togel Data', 'gambling', 'keluaran togel|prediksi togel|data sgp|data hk|data sdy|pengeluaran sgp|pengeluaran hk|result togel|angka main|angka jitu|angka keluar|result hk|result sgp', 'regex', 'critical', 'Deteksi data togel', TRUE),
('Togel Live Draw', 'gambling', 'live draw sgp|live draw hk|live draw sdy|paito warna|paito sgp|paito hk|rumus togel|syair togel|bocoran togel|prediksi jitu|ekor jitu', 'regex', 'critical', 'Deteksi live draw togel', TRUE),
('Toto 4D', 'gambling', 'toto gelap|toto macau|toto sgp|toto hk|toto4d|togel4d|4dprize|angka4d|colok bebas|colok jitu|colok naga|colok macau|shio togel', 'regex', 'critical', 'Deteksi toto 4D', TRUE),
('Togel Bandar', 'gambling', 'bandar togel|agen togel|bo togel|bandar toto|agen toto|bandar colok|bandar 4d|bandar pools', 'regex', 'critical', 'Deteksi bandar togel', TRUE),

-- Togel Site Names (from blocklist)
('Togel Sites', 'gambling', 'wargatogel|iontogel|nenektogel|kakektogel|indotogel|asiatogel|eurotogel|royaltogel|togelcc|togel178|togel158|togel279|totobet|totojitu|totokl', 'regex', 'critical', 'Deteksi nama situs togel', TRUE),

-- Casino & Poker
('Casino Online', 'gambling', 'casino online|live casino|baccarat|blackjack|roulette|sic bo|dragon tiger|sexy baccarat|sa gaming|evolution gaming|dream gaming|ag casino|wm casino', 'regex', 'critical', 'Deteksi kata kunci casino online', TRUE),
('Poker QQ', 'gambling', 'poker online|domino qq|dominoqq|bandarq|bandarqq|capsa susun|pkv games|pkvgames|idnpoker|idnplay|poker88|dewapoker|nagapoker', 'regex', 'critical', 'Deteksi poker dan QQ games', TRUE),
('Card Games', 'gambling', 'ceme online|ceme keliling|super10|omaha poker|texas holdem|samgong|gaple|sakong', 'regex', 'critical', 'Deteksi permainan kartu online', TRUE),

-- Sports Betting
('Judi Bola', 'gambling', 'judi bola|taruhan bola|bandar bola|agen bola|sbobet|sbobet88|maxbet|ibcbet|cmdbet|parlay|mix parlay|taruhan sepakbola', 'regex', 'critical', 'Deteksi judi bola', TRUE),
('Sports Betting', 'gambling', 'sportsbook|handicap|over under|asian handicap|pasaran bola|odds bola|prediksi bola|tips bola|betting tips|livescore', 'regex', 'critical', 'Deteksi sports betting', TRUE),
('E-Sports Betting', 'gambling', 'esports betting|taruhan esports|dota betting|csgo betting|valorant betting|mobile legend betting', 'regex', 'high', 'Deteksi e-sports betting', TRUE),

-- Bonus & Promo
('Bonus Judi', 'gambling', 'bonus new member|bonus deposit|bonus cashback|bonus referral|freebet gratis|bonus 100%|depo 25 bonus 25|depo 50 bonus 50|bonus harian|bonus mingguan', 'regex', 'high', 'Deteksi bonus dan promosi judi', TRUE),
('Deposit Promo', 'gambling', 'deposit 10rb|deposit 20rb|deposit minimal|minimal depo|tanpa potongan|wd cepat|proses cepat|depo pulsa|depo dana|depo 5000|depo 10000|depo 25000', 'regex', 'high', 'Deteksi promo deposit', TRUE),
('Promo Keywords', 'gambling', 'promo member baru|cashback slot|cashback togel|turnover slot|rollingan|rebate|event jackpot|jackpot harian', 'regex', 'high', 'Deteksi promo keywords', TRUE),

-- Registration Keywords
('Daftar Judi', 'gambling', 'daftar slot|daftar togel|daftar casino|daftar poker|daftar judi|link alternatif|link resmi|link daftar|akun demo|akun pro|daftar akun|buat akun', 'regex', 'critical', 'Deteksi link pendaftaran judi', TRUE),
('Situs Terpercaya', 'gambling', 'situs terpercaya|bandar terpercaya|agen resmi|bo terpercaya|situs gacor|situs jp|situs resmi|terpercaya 2024|terpercaya 2025', 'regex', 'high', 'Deteksi kata kunci situs terpercaya', TRUE),
('Login Keywords', 'gambling', 'login slot|login togel|login casino|login poker|login judi|login member|member area|masuk member', 'regex', 'high', 'Deteksi kata kunci login', TRUE),

-- Slang Judol (Indonesian Gambling Slang)
('Slang Judol', 'gambling', 'judol|jp paus|auto sultan|auto cuan|sensational|gacor parah|cuan gede|jepe|pecah jp|pola gacor|jam gacor|waktu gacor|spin paus', 'regex', 'critical', 'Deteksi slang judol Indonesia', TRUE),
('Slang Judi Extended', 'gambling', 'panen jp|panen cuan|sultan mode|mode sultan|gaceng|wibu jp|bocil slot|gas terus|santuy jp|mantap jiwa|wd besar|wd gede', 'regex', 'high', 'Deteksi slang judi lanjutan', TRUE),
('Prosperity Keywords', 'gambling', 'pasti dapat|menang terus|menang besar|kaya raya|auto kaya|jadi sultan|jadi miliarder|passive income slot|penghasilan tambahan', 'regex', 'high', 'Deteksi kata-kata prosperity (dari GbDetector)', TRUE),

-- Payment Methods
('Deposit Pulsa', 'gambling', 'deposit pulsa|depo pulsa|pulsa tanpa potongan|pulsa telkomsel|pulsa xl|pulsa indosat|via pulsa|slot pulsa', 'regex', 'high', 'Deteksi deposit via pulsa', TRUE),
('Deposit E-Wallet', 'gambling', 'deposit dana|depo dana|deposit ovo|depo ovo|deposit gopay|depo gopay|deposit linkaja|qris slot|qris judi|deposit shopeepay', 'regex', 'high', 'Deteksi deposit via e-wallet', TRUE),
('Deposit Bank', 'gambling', 'deposit bca|deposit mandiri|deposit bni|deposit bri|transfer bank|via bank|rekening slot', 'regex', 'high', 'Deteksi deposit via bank', TRUE),
('Crypto Deposit', 'gambling', 'deposit crypto|deposit usdt|deposit bitcoin|deposit ethereum|crypto slot|usdt slot|btc slot', 'regex', 'high', 'Deteksi deposit crypto', TRUE),

-- URL Patterns Extended
('Gambling URL Pattern', 'gambling', '(slot|togel|casino|poker|judi|betting|gacor|maxwin|cuan|hoki|raja|dewa|boss|mpo|win|jp|bet|toto)\\d*\\.(com|net|org|io|site|online|xyz|club|vip|pro|bet|games|fun|live|sbs|lol|cyou|shop|store|cc|co)', 'regex', 'critical', 'Deteksi pola URL situs judi', TRUE),
('Gambling Subdomain', 'gambling', '(slot|togel|casino|poker|judi|gacor|maxwin|cuan|bet|toto)\\d*\\.\\w+\\.(com|net|org|id|co\\.id|go\\.id|ac\\.id)', 'regex', 'critical', 'Deteksi subdomain judi', TRUE),
('Redirect Domain Pattern', 'gambling', '(bit\\.ly|tinyurl|s\\.id|cutt\\.ly|rebrand\\.ly|shorturl).*?(slot|togel|judi|gacor|casino)', 'regex', 'high', 'Deteksi short URL judi', TRUE),

-- Redirect Scripts
('Redirect Judi', 'gambling', '(window\\.location|location\\.href|location\\.replace|document\\.location)\\s*=\\s*[\'\"](https?://)?[^\'\"]*?(slot|togel|casino|judi|poker|gacor)', 'regex', 'critical', 'Deteksi script redirect ke situs judi', TRUE),
('Meta Refresh Redirect', 'gambling', '<meta[^>]*http-equiv\\s*=\\s*[\"'']refresh[\"''][^>]*url\\s*=\\s*[^>]*?(slot|togel|casino|judi|gacor)', 'regex', 'critical', 'Deteksi meta refresh redirect', TRUE),

-- Hidden Content Detection
('Hidden Gambling Link', 'gambling', '<a[^>]*href\\s*=\\s*[\"''][^\"'']*?(slot|togel|casino|judi|poker|gacor)[^\"'']*?[\"''][^>]*(display:\\s*none|visibility:\\s*hidden|opacity:\\s*0)', 'regex', 'high', 'Deteksi link judi tersembunyi', TRUE),
('Hidden Text Position', 'gambling', '(position:\\s*absolute|left:\\s*-\\d{4,}px|top:\\s*-\\d{4,}px|text-indent:\\s*-\\d{4,})[^}]*?(slot|togel|judi|gacor|casino)', 'regex', 'high', 'Deteksi teks tersembunyi dengan posisi', TRUE),
('Hidden Iframe', 'gambling', '<iframe[^>]*?(slot|togel|judi|gacor|casino)[^>]*?(display:\\s*none|visibility:\\s*hidden|width:\\s*0|height:\\s*0)', 'regex', 'critical', 'Deteksi iframe judi tersembunyi', TRUE),

-- Meta Tags Detection
('Gambling Meta', 'gambling', '<meta[^>]*(content|name)\\s*=\\s*[\"''][^\"'']*?(slot|togel|casino|judi|poker|gacor|maxwin|jackpot)[^\"'']*?[\"'']', 'regex', 'high', 'Deteksi meta tag judi', TRUE),
('Gambling Title', 'gambling', '<title>[^<]*?(slot gacor|togel online|judi online|casino online|poker online|maxwin|jackpot)[^<]*?</title>', 'regex', 'critical', 'Deteksi title tag judi', TRUE),

-- Folder/Path Patterns (from CSIRT guide)
('Slot Gacor Folder', 'gambling', '(/|\\\\)(slot-?gacor|judi-?online|togel-?online|casino-?online|poker-?online)(/|\\\\|$)', 'regex', 'critical', 'Deteksi folder/path slot gacor', TRUE),

-- Image/Media Patterns
('Gambling Image Alt', 'gambling', '<img[^>]*alt\\s*=\\s*[\"''][^\"'']*?(slot|togel|judi|gacor|casino|maxwin|jackpot)[^\"'']*?[\"'']', 'regex', 'high', 'Deteksi alt image judi', TRUE),

-- =============================================
-- DEFACEMENT PATTERNS
-- =============================================
('Hacked By', 'defacement', 'hacked by|defaced by|owned by|pwned by|h4ck[e3]d|d[e3]f[a4]c[e3]d', 'regex', 'critical', 'Deteksi pesan defacement standar', TRUE),
('Indonesian Defacer', 'defacement', 'indonesian defacer|garuda cyber|indonesia cyber|jatim cyber|surabaya black hat|jakarta cyber|cyber indonesia', 'regex', 'critical', 'Deteksi defacer Indonesia', TRUE),
('Defacement Message', 'defacement', 'this site has been hacked|website hacked|security breach|we are anonymous|expect us', 'regex', 'critical', 'Pesan defacement umum', TRUE),
('Zone-H Mirror', 'defacement', 'zone-?h\\.(com|org)|mirror|notified\\s*by', 'regex', 'critical', 'Deteksi referensi Zone-H', TRUE),

-- =============================================
-- WEBSHELL PATTERNS
-- =============================================
('C99 Shell', 'webshell', 'c99shell|c99\\.php|c99_shell', 'regex', 'critical', 'Deteksi webshell C99', TRUE),
('R57 Shell', 'webshell', 'r57shell|r57\\.php|r57_shell', 'regex', 'critical', 'Deteksi webshell R57', TRUE),
('B374K Shell', 'webshell', 'b374k|b374k\\.php', 'regex', 'critical', 'Deteksi webshell B374K', TRUE),
('WSO Shell', 'webshell', 'wso shell|wso\\.php|web shell by', 'regex', 'critical', 'Deteksi webshell WSO', TRUE),
('PHP Shell Generic', 'webshell', 'FilesMan|Alfa Shell|Priv8|IndoXploit|mini shell|php spy', 'regex', 'critical', 'Deteksi webshell PHP umum', TRUE),
('Shell Functions', 'webshell', 'eval\\s*\\(\\s*\\$_(GET|POST|REQUEST|COOKIE)|shell_exec|passthru|system\\s*\\(\\s*\\$_', 'regex', 'critical', 'Deteksi fungsi webshell', TRUE),

-- =============================================
-- MALWARE PATTERNS
-- =============================================
('Cryptominer', 'malware', 'coinhive|cryptoloot|coin-?hive|minero|webminer|miner\\.start', 'regex', 'critical', 'Deteksi cryptominer script', TRUE),
('Malicious Iframe', 'malware', '<iframe[^>]*?(display:\\s*none|visibility:\\s*hidden|width:\\s*[01]px|height:\\s*[01]px)', 'regex', 'critical', 'Deteksi iframe tersembunyi', TRUE),
('Suspicious Redirect', 'malware', 'eval\\(.*unescape|document\\.write\\(unescape|window\\.location.*=.*http', 'regex', 'high', 'Deteksi redirect mencurigakan', TRUE),
('Base64 Encoded', 'malware', 'eval\\(base64_decode|atob\\(|fromCharCode', 'regex', 'high', 'Deteksi kode base64', TRUE),

-- =============================================
-- PHISHING PATTERNS
-- =============================================
('Login Phishing', 'phishing', 'verify your account|confirm your identity|update your information|akun anda diblokir', 'regex', 'high', 'Deteksi phishing login', TRUE),
('Bank Phishing', 'phishing', 'bank central asia|bank mandiri|bank bri|bank bni|token internet banking|verifikasi akun bank', 'regex', 'critical', 'Deteksi phishing bank Indonesia', TRUE),
('E-wallet Phishing', 'phishing', 'verifikasi gopay|verifikasi ovo|verifikasi dana|klaim saldo', 'regex', 'high', 'Deteksi phishing e-wallet', TRUE),

-- =============================================
-- SEO SPAM PATTERNS
-- =============================================
('SEO Spam Links', 'seo_spam', 'buy cheap|viagra|cialis|casino bonus|payday loan', 'regex', 'medium', 'Deteksi SEO spam', TRUE),
('Hidden Text Spam', 'seo_spam', 'color:\\s*(#fff|white|transparent)|font-size:\\s*0|text-indent:\\s*-9999', 'regex', 'medium', 'Deteksi teks tersembunyi', TRUE),
('Japanese SEO Spam', 'seo_spam', '[\\x{3040}-\\x{309F}\\x{30A0}-\\x{30FF}]{10,}', 'regex', 'high', 'Deteksi spam Jepang', TRUE),

-- =============================================
-- BACKDOOR PATTERNS
-- =============================================
('PHP Backdoor', 'backdoor', 'shell_exec|passthru|system\\(|exec\\(.*\\$_|eval\\(.*\\$_', 'regex', 'critical', 'Deteksi backdoor PHP', TRUE),
('File Upload Backdoor', 'backdoor', 'move_uploaded_file.*\\$_FILES|file_put_contents.*\\$_REQUEST', 'regex', 'critical', 'Deteksi backdoor upload', TRUE),
('Remote Include', 'backdoor', '(include|require)\\s*\\(\\s*[\"'']?(https?://|\\$_(GET|POST|REQUEST))', 'regex', 'critical', 'Deteksi remote file inclusion', TRUE),

-- =============================================
-- INJECTION PATTERNS
-- =============================================
('SQL Injection', 'injection', 'UNION\\s+SELECT|DROP\\s+TABLE|INSERT\\s+INTO|DELETE\\s+FROM|information_schema', 'regex', 'critical', 'Deteksi SQLi', TRUE),
('XSS Payload', 'injection', '<script>alert|onerror=|onload=|javascript:|onclick=', 'regex', 'high', 'Deteksi XSS payload', TRUE);
