package domain

import (
	"time"
)

// DorkCategory represents the category of dork pattern
type DorkCategory string

const (
	DorkCategoryGambling   DorkCategory = "gambling"
	DorkCategoryDefacement DorkCategory = "defacement"
	DorkCategoryMalware    DorkCategory = "malware"
	DorkCategoryPhishing   DorkCategory = "phishing"
	DorkCategorySEOSpam    DorkCategory = "seo_spam"
	DorkCategoryShell      DorkCategory = "webshell"
	DorkCategoryBackdoor   DorkCategory = "backdoor"
	DorkCategoryInjection  DorkCategory = "injection"
)

// DorkSeverity represents the severity level of detection
type DorkSeverity string

const (
	DorkSeverityCritical DorkSeverity = "critical"
	DorkSeverityHigh     DorkSeverity = "high"
	DorkSeverityMedium   DorkSeverity = "medium"
	DorkSeverityLow      DorkSeverity = "low"
)

// DorkPattern represents a detection pattern
type DorkPattern struct {
	ID          int64        `db:"id" json:"id"`
	Category    DorkCategory `db:"category" json:"category"`
	Name        string       `db:"name" json:"name"`
	Pattern     string       `db:"pattern" json:"pattern"`             // Regex pattern
	PatternType string       `db:"pattern_type" json:"pattern_type"`   // keyword, regex, xpath, css_selector
	Keywords    []string     `json:"keywords"`                         // Keywords to search
	Description string       `db:"description" json:"description"`
	Severity    DorkSeverity `db:"severity" json:"severity"`
	IsActive    bool         `db:"is_active" json:"is_active"`
	IsRegex     bool         `db:"is_regex" json:"is_regex"`
	IsDefault   bool         `db:"is_default" json:"is_default"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at" json:"updated_at,omitempty"`
}

// DorkScanResult represents the result of a dork scan
type DorkScanResult struct {
	ID                int64           `db:"id" json:"id"`
	WebsiteID         int64           `db:"website_id" json:"website_id"`
	WebsiteName       string          `json:"website_name,omitempty"`
	WebsiteURL        string          `json:"website_url,omitempty"`
	ScanType          string          `db:"scan_type" json:"scan_type"` // full, quick, targeted
	Status            string          `db:"status" json:"status"`       // pending, running, completed, failed
	TotalPagesScanned int             `db:"total_pages_scanned" json:"total_pages_scanned"`
	TotalPatterns     int             `db:"total_patterns" json:"total_patterns"`
	MatchedPatterns   int             `db:"matched_patterns" json:"matched_patterns"`
	TotalDetections   int             `db:"total_detections" json:"total_detections"`
	CriticalCount     int             `db:"critical_count" json:"critical_count"`
	HighCount         int             `db:"high_count" json:"high_count"`
	MediumCount       int             `db:"medium_count" json:"medium_count"`
	LowCount          int             `db:"low_count" json:"low_count"`
	AIFilteredCount   int             `db:"ai_filtered_count" json:"ai_filtered_count"`
	CategoriesScanned []DorkCategory  `json:"categories_scanned,omitempty"`
	ThreatLevel       DorkSeverity    `db:"threat_level" json:"threat_level"`
	Detections        []DorkDetection `json:"detections,omitempty"`
	ScanDuration      int64           `db:"scan_duration" json:"scan_duration_ms"`
	StartedAt         *time.Time      `db:"started_at" json:"started_at,omitempty"`
	CompletedAt       *time.Time      `db:"completed_at" json:"completed_at,omitempty"`
	ErrorMessage      string          `db:"error_message" json:"error_message,omitempty"`
	ScannedAt         time.Time       `db:"scanned_at" json:"scanned_at"`
	CreatedAt         time.Time       `db:"created_at" json:"created_at"`
}

// DorkDetection represents a single detection
type DorkDetection struct {
	ID              int64        `db:"id" json:"id"`
	ScanResultID    int64        `db:"scan_result_id" json:"scan_result_id"`
	WebsiteID       int64        `db:"website_id" json:"website_id"`
	WebsiteName     string       `json:"website_name,omitempty"`
	WebsiteURL      string       `json:"website_url,omitempty"`
	PatternID       int64        `db:"pattern_id" json:"pattern_id"`
	Category        DorkCategory `db:"category" json:"category"`
	PatternName     string       `db:"pattern_name" json:"pattern_name"`
	Severity        DorkSeverity `db:"severity" json:"severity"`
	URL             string       `db:"url" json:"url"`
	MatchedContent  string       `db:"matched_content" json:"matched_content"`
	MatchedText     string       `db:"matched_text" json:"matched_text,omitempty"` // Alias for backward compatibility
	Context         string       `db:"context" json:"context,omitempty"`
	Location        string       `db:"location" json:"location,omitempty"` // URL path, HTML element, etc.
	Confidence      float64      `db:"confidence" json:"confidence"`
	AIVerified      bool         `db:"ai_verified" json:"ai_verified"`
	IsFalsePositive bool         `db:"is_false_positive" json:"is_false_positive"`
	IsResolved      bool         `db:"is_resolved" json:"is_resolved"`
	ResolvedAt      *time.Time   `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy      string       `db:"resolved_by" json:"resolved_by,omitempty"`
	Notes           string       `db:"notes" json:"notes,omitempty"`
	DetectedAt      time.Time    `db:"detected_at" json:"detected_at"`
	CreatedAt       time.Time    `db:"created_at" json:"created_at"`
}

// DorkScanRequest represents a request to scan
type DorkScanRequest struct {
	WebsiteID  int64          `json:"website_id"`
	URL        string         `json:"url"`
	Categories []DorkCategory `json:"categories,omitempty"` // Empty means all
	ScanType   string         `json:"scan_type"`            // full, quick, targeted
	Depth      int            `json:"depth"`                // How many pages to crawl
}

// DorkScanSummary represents summary of all scans
type DorkScanSummary struct {
	TotalScans        int            `json:"total_scans"`
	ThreatsDetected   int            `json:"threats_detected"`
	CriticalThreats   int            `json:"critical_threats"`
	HighThreats       int            `json:"high_threats"`
	MediumThreats     int            `json:"medium_threats"`
	LowThreats        int            `json:"low_threats"`
	WebsitesAffected  int            `json:"websites_affected"`
	TopCategories     []CategoryStat `json:"top_categories"`
	RecentDetections  []DorkDetection `json:"recent_detections"`
	LastScanAt        *time.Time     `json:"last_scan_at,omitempty"`
}

// CategoryStat represents statistics by category
type CategoryStat struct {
	Category DorkCategory `json:"category"`
	Count    int          `json:"count"`
	Severity DorkSeverity `json:"highest_severity"`
}

// Predefined dork patterns for Indonesian government websites
// Source: arxiv.org/html/2508.19368v1, GbDetector (GitHub), BrigsLabs/judol blocklist, CSIRT guides
var DefaultDorkPatterns = []DorkPattern{
	// === GAMBLING (JUDOL) PATTERNS ===
	// Based on research: Keywords with lowest false positive rate

	// Core Detection Keywords (from arxiv research - most reliable)
	{Category: DorkCategoryGambling, Name: "Core Detection Keywords", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"togel", "toto", "judi", "slot", "gacor", "bandar", "maxwin", "zeus", "judol",
		},
		Description: "Kata kunci inti deteksi judi (penelitian arxiv - false positive rendah)"},

	// Slot Keywords - Basic
	{Category: DorkCategoryGambling, Name: "Slot Online Keywords", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"slot gacor", "slot online", "slot88", "slot777", "slot138", "slot212",
			"slot deposit pulsa", "slot dana", "slot ovo", "slot gopay", "slot linkaja",
			"rtp slot", "rtp live", "rtp gacor", "bocoran slot", "slot maxwin", "slot jackpot", "demo slot",
			"slot gratis", "akun slot", "akun pro", "akun vip", "slot server",
			"slot thailand", "slot luar negeri", "slot anti rungkad", "slot anti lag",
			"slot pasti jp", "slot gampang menang", "slot hoki", "slot4d", "slot5000",
			"scatter hitam", "scatter petir", "sensational", "slot demo", "slot pragmatic",
		},
		Description: "Kata kunci slot online umum"},

	// Slot Provider - Pragmatic Play (40% market share Indonesia)
	{Category: DorkCategoryGambling, Name: "Pragmatic Play Games", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"pragmatic play", "pragmatic88", "pragmaticplay", "pp slot",
			"gates of olympus", "sweet bonanza", "starlight princess", "wild west gold",
			"aztec gems", "great rhino", "wolf gold", "john hunter", "mustang gold",
			"release the kraken", "buffalo king", "hot fiesta", "fruit party",
			"the dog house", "bigger bass bonanza", "floating dragon", "madame destiny",
			"gems bonanza", "power of thor", "gates of aztec", "gates of gatot kaca",
			"kakek zeus", "dewa zeus", "sugar rush", "big bass splash",
		},
		Description: "Game dari Pragmatic Play"},

	// Slot Provider - PG Soft (25% market share Indonesia)
	{Category: DorkCategoryGambling, Name: "PG Soft Games", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"pg soft", "pgsoft", "pg slot", "mahjong ways", "mahjong ways 2", "mahjong wins",
			"fortune tiger", "fortune ox", "fortune mouse", "fortune rabbit", "fortune dragon",
			"lucky neko", "dragon tiger luck", "treasures of aztec", "ganesha fortune",
			"leprechaun riches", "ninja vs samurai", "phoenix rises", "wild bandito",
			"candy bonanza", "crypto gold", "double fortune", "emperor's favour",
			"flirting scholar", "hood vs wolf", "jungle delight", "legend of perseus",
		},
		Description: "Game dari PG Soft"},

	// Slot Provider - Others
	{Category: DorkCategoryGambling, Name: "Slot Provider Others", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"joker123", "joker388", "joker gaming", "habanero", "spadegaming", "microgaming",
			"playtech", "cq9", "jdb", "afb gaming", "live22", "ace333", "mega888",
			"pussy888", "xe88", "918kiss", "kiss918", "scr888", "newtown", "play1628",
			"rollex11", "red tiger", "isoftbet", "yggdrasil", "netent", "betsoft",
			"nolimit city", "relax gaming", "hacksaw gaming", "push gaming",
			"jili", "fachai", "evoplay", "booongo", "playson", "wazdan",
		},
		Description: "Nama-nama provider slot lainnya"},

	// Popular Gambling Site Names - Comprehensive (with leetspeak variations)
	{Category: DorkCategoryGambling, Name: "Situs Judi Populer", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			// Raja series
			"rajabola", "rajabol4", "rajab0la", "raj4bola", "rajacuan", "rajaslot",
			"rajaslot88", "rajatogel", "rajat0gel", "rajapoker", "rajacasino", "rajabet",
			"raja388", "raja189", "raja69", "raja99", "rajawin", "rajajp", "rajahoki",
			// Dewa series
			"dewabet", "dewa288", "dewacasino", "dewapoker", "dewatogel", "dewajudi",
			"dewa303", "dewa234", "dewacash", "dewaslot", "dewawin", "dewahoki",
			"dewa88", "dewa99", "dewacuan",
			// Boss series
			"bos88", "bosku", "bosswin", "bosslot", "bosjp", "bostogel", "bosqq",
			"bosbet", "boss88", "boscuan", "bosshoki", "bos303", "bos99", "bosjudi",
			// Mpo series
			"mpo88", "mpo100", "mpo188", "mpo189", "mpo500", "mpo777", "mpo868",
			"mposlot", "mpoplay", "mpocash", "mpo388", "mpo555", "mpo99", "mpo303",
			"mpobos", "mpohoki", "mpowin",
			// Gacor series
			"gacor88", "gacor77", "gacor123", "gacorjp", "gacor4d", "gacor188",
			"gacorbet", "gacorwin", "slotgacor", "gacor303", "gacormania", "gacortoto",
			// Hoki/Lucky series
			"hoki88", "hoki99", "hoki777", "hokibet", "hokijudi", "hokislot",
			"lucky88", "lucky77", "luckybet", "lucky303", "hokicuan", "hokijp",
			// Win series
			"win88", "win99", "win138", "win303", "winbet", "winjudi", "menang88",
			"menang123", "menangbet", "menangslot", "wintoto", "wincuan", "winhoki",
			// JP (Jackpot) series
			"jp88", "jp99", "jp138", "jp303", "jpslot", "jptogel", "jp4d",
			"jackpot88", "jackpot99", "jackpotslot", "jphoki", "jpwin", "jpcuan",
			// Cuan series
			"cuan88", "cuan99", "cuan123", "cuanbet", "cuanslot", "cuanjp",
			"cuantogel", "cuan4d", "cuan303", "cuanwin", "cuanhoki", "datacuan",
			// 88 series (very common pattern)
			"indo88", "asia88", "emas88", "uang88", "rupiah88", "duit88",
			"bola88", "casino88", "poker88", "togel88", "toto88", "naga88",
			"dragon88", "sultan88", "mega88", "super88", "power88", "hot88",
			"vip88", "royal88", "zeus88", "olympus88",
			// 303 series
			"asia303", "indo303", "euro303", "vegas303", "naga303", "dragon303",
			"slot303", "togel303", "casino303", "sultan303", "mega303", "royal303",
			// Popular brands
			"sbobet", "sbobet88", "sbo88", "maxbet", "ibcbet", "cmdbet",
			"m88", "fun88", "w88", "12bet", "188bet", "bet365", "1xbet",
			// IDN series
			"idnpoker", "idnplay", "idnslot", "idntogel", "idncasino", "idn88",
			// PKV series
			"pkvgames", "pkv88", "pkvpoker", "pkvqq",
			// QQ series
			"dominoqq", "bandarqq", "pokerqq", "aduqq", "sakongqq", "qq88", "qqslot",
			// 4D series
			"slot4d", "togel4d", "toto4d", "judi4d", "situs4d", "hoki4d", "jp4d",
			"cuan4d", "win4d", "play4d", "vip4d", "mega4d",
			// Naga series
			"naga88", "naga303", "nagaslot", "nagatogel", "nagapoker", "nagabet",
			// Dragon series
			"dragon88", "dragon303", "dragonslot", "dragontogel", "dragonbet",
			// Sultan series
			"sultan88", "sultan303", "sultanslot", "sultantogel", "sultanbet", "sultanplay",
			// Misc popular from blocklist
			"alexistogel", "bandar55", "ligaciputra", "panen88", "receh88",
			"warungbet", "warung88", "kakekpro", "macanslot", "sensaslot",
		},
		Description: "Nama-nama situs judi populer Indonesia"},

	// Togel Site Names (from BrigsLabs/judol blocklist)
	{Category: DorkCategoryGambling, Name: "Situs Togel Populer", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"wargatogel", "iontogel", "nenektogel", "kakektogel", "indotogel",
			"asiatogel", "eurotogel", "royaltogel", "togelcc", "togel178",
			"togel158", "togel279", "totobet", "totojitu", "totokl",
			"jangkartoto", "rubah4d", "lipat4d", "wargatoto", "sigmaslot",
			"bandarcolok", "bandartogel", "agencolok", "bandartoto",
		},
		Description: "Nama situs togel dari blocklist"},

	// Leetspeak/Number Substitution Patterns (GbDetector-style)
	{Category: DorkCategoryGambling, Name: "Leetspeak Gambling Names", Severity: DorkSeverityCritical, IsRegex: true, IsActive: true,
		Pattern:     `(?i)(r[a4]j[a4]|d[e3]w[a4]|b[o0]s|h[o0]k[i1]|w[i1]n|j[a4]ckp[o0]t|g[a4]c[o0]r|cu[a4]n|m[a4]xw[i1]n)(b[o0]l[a4]|sl[o0]t|t[o0]g[e3]l|c[a4]s[i1]n[o0]|p[o0]k[e3]r|b[e3]t|jud[i1]|qq)?\d*`,
		Description: "Deteksi nama judi dengan variasi leetspeak"},

	// Character Evasion Detection (from GbDetector)
	{Category: DorkCategoryGambling, Name: "Evasion Dot Separated", Severity: DorkSeverityHigh, IsRegex: true, IsActive: true,
		Pattern:     `(?i)s\.l\.o\.t|t\.o\.g\.e\.l|j\.u\.d\.i|g\.a\.c\.o\.r|c\.a\.s\.i\.n\.o`,
		Description: "Deteksi evasion dengan titik pemisah"},
	{Category: DorkCategoryGambling, Name: "Evasion Space Separated", Severity: DorkSeverityHigh, IsRegex: true, IsActive: true,
		Pattern:     `(?i)s\s+l\s+o\s+t|t\s+o\s+g\s+e\s+l|j\s+u\s+d\s+i|g\s+a\s+c\s+o\s+r`,
		Description: "Deteksi evasion dengan spasi pemisah"},
	{Category: DorkCategoryGambling, Name: "Evasion Star Separated", Severity: DorkSeverityHigh, IsRegex: true, IsActive: true,
		Pattern:     `(?i)s\*l\*o\*t|t\*o\*g\*e\*l|j\*u\*d\*i|g\*a\*c\*o\*r`,
		Description: "Deteksi evasion dengan asterisk pemisah"},
	{Category: DorkCategoryGambling, Name: "Evasion Dash Separated", Severity: DorkSeverityHigh, IsRegex: true, IsActive: true,
		Pattern:     `(?i)s-l-o-t|t-o-g-e-l|j-u-d-i|g-a-c-o-r|c-a-s-i-n-o`,
		Description: "Deteksi evasion dengan dash pemisah"},

	// Indonesian Prosperity/Winning Keywords (from GbDetector)
	{Category: DorkCategoryGambling, Name: "Prosperity Keywords", Severity: DorkSeverityHigh, IsRegex: false, IsActive: true,
		Keywords: []string{
			"menang", "gacor", "senang", "gembira", "kaya", "pasti dapat",
			"bangga", "panen", "cuan gede", "auto kaya", "auto sultan",
			"jadi miliarder", "passive income", "penghasilan tambahan",
			"kaya raya", "rezeki nomplok", "hoki terus", "jp terus",
		},
		Description: "Kata kunci prosperity/kemenangan (dari GbDetector)"},

	// Togel Keywords - Extended
	{Category: DorkCategoryGambling, Name: "Togel Keywords Extended", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"togel", "togel online", "togel singapore", "togel hongkong", "togel sydney",
			"togel sgp", "togel hk", "togel sdy", "togel macau", "togel cambodia",
			"togel china", "togel japan", "togel taiwan", "togel korea",
			"keluaran togel", "prediksi togel", "bandar togel", "agen togel",
			"togel hari ini", "data sgp", "data hk", "data sdy", "data sidney",
			"pengeluaran sgp", "pengeluaran hk", "result togel", "angka main",
			"angka jitu", "angka keramat", "rumus togel", "syair togel",
			"paito warna", "paito sgp", "paito hk", "live draw sgp", "live draw hk",
			"toto gelap", "toto macau", "toto sgp", "toto hk", "toto sdy",
			"colok bebas", "colok jitu", "colok naga", "colok macau",
			"togel 2d", "togel 3d", "togel 4d", "prize 2d", "prize 3d", "prize 4d", "shio togel", "ekor togel", "kepala togel",
			"bbfs", "tardal", "invest togel", "ai togel", "ck togel",
		},
		Description: "Kata kunci togel lengkap"},

	// Casino Keywords - Extended
	{Category: DorkCategoryGambling, Name: "Casino Keywords Extended", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"casino online", "live casino", "baccarat", "blackjack", "roulette",
			"sic bo", "dragon tiger", "poker online", "domino qq", "bandar ceme",
			"capsa susun", "super10", "omaha", "texas holdem", "gaple",
			"bandar poker", "bandar qq", "bandar sakong", "aduq", "perang baccarat",
			"fantan", "keno", "casino war", "three card poker", "pai gow",
			"sexy baccarat", "dream gaming", "ag casino", "sa gaming", "ebet",
			"evolution gaming", "pragmatic live", "ezugi", "allbet", "wm casino",
			"asia gaming", "og casino", "opus gaming", "gold deluxe",
		},
		Description: "Kata kunci casino online lengkap"},

	// Sports Betting Keywords - Extended
	{Category: DorkCategoryGambling, Name: "Sports Betting Extended", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"sbobet", "maxbet", "ibcbet", "cmdbet", "citibet", "m8bet",
			"taruhan bola", "judi bola", "bandar bola", "agen bola",
			"parlay", "mix parlay", "parlay bola", "parlay jitu",
			"handicap", "over under", "odds", "pasaran bola",
			"sportsbook", "livescore betting", "prediksi bola", "tips bola",
			"taruhan olahraga", "basket betting", "tenis betting", "esport betting",
			"virtual sport", "virtual football", "number game", "keno sport",
			"bursa taruhan", "odds bola", "asian handicap", "1x2",
		},
		Description: "Kata kunci taruhan bola lengkap"},

	// Bonus & Promotion Keywords
	{Category: DorkCategoryGambling, Name: "Bonus Promotion Keywords", Severity: DorkSeverityHigh, IsRegex: false, IsActive: true,
		Keywords: []string{
			"bonus new member", "bonus deposit", "bonus cashback", "bonus referral",
			"bonus harian", "bonus mingguan", "bonus bulanan", "bonus rollingan",
			"bonus turnover", "bonus rebate", "bonus freebet", "freebet gratis",
			"promo slot", "promo togel", "promo casino", "promo sportsbook",
			"depo 10 bonus 10", "depo 25 bonus 25", "depo 50 bonus 50",
			"deposit 10rb", "deposit 20rb", "deposit 25rb", "deposit 50rb",
			"minimal deposit", "minimal withdraw", "depo pulsa", "depo ewallet",
			"tanpa potongan", "tanpa ribet", "proses cepat", "wd cepat",
			"bonus garansi kekalahan", "bonus next deposit", "welcome bonus",
		},
		Description: "Kata kunci bonus dan promosi judi"},

	// Registration & Login Keywords
	{Category: DorkCategoryGambling, Name: "Daftar Login Judi", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"daftar slot", "daftar togel", "daftar casino", "daftar poker",
			"daftar judi", "daftar akun", "daftar gratis", "daftar sekarang",
			"login slot", "login togel", "login casino", "login poker",
			"link alternatif", "link resmi", "link daftar", "link login",
			"rtp live", "rtp hari ini", "info rtp", "bocoran rtp",
			"akun demo", "akun pro", "akun vip", "akun jp",
			"situs gacor", "situs jp",
			"bandar terpercaya", "agen resmi", "agen terpercaya",
		},
		Description: "Kata kunci pendaftaran dan login judi"},

	// Indonesian Gambling Slang
	{Category: DorkCategoryGambling, Name: "Slang Judol Indonesia", Severity: DorkSeverityCritical, IsRegex: false, IsActive: true,
		Keywords: []string{
			"judol", "judi online", "bandar judi", "agen judi", "situs judi",
			"bo terpercaya", "bo slot", "bo togel", "bo casino",
			"wd lancar", "depo cepat", "jp paus", "auto sultan", "auto cuan",
			"sensational", "gacor parah", "lagi gacor", "mantap jiwa",
			"cuan gede", "jepe", "jp besar", "pecah jp",
			"scatter hitam", "scatter petir", "zeus gacor", "olympus gacor",
			"mahjong ways", "gates of olympus", "starlight princess", "sweet bonanza",
			"wild west gold", "aztec gems", "bonanza gold", "great rhino",
			"koi gate", "lucky neko", "fortune tiger", "fortune rabbit",
			"pola gacor", "pola slot", "pola jp", "trik slot", "cheat slot",
			"jam gacor", "jam hoki", "waktu gacor", "prime time slot",
		},
		Description: "Slang dan istilah judol Indonesia"},

	// Payment Methods for Gambling
	{Category: DorkCategoryGambling, Name: "Metode Pembayaran Judi", Severity: DorkSeverityHigh, IsRegex: false, IsActive: true,
		Keywords: []string{
			"deposit pulsa", "depo pulsa", "pulsa tanpa potongan",
			"deposit dana", "depo dana", "deposit ovo", "depo ovo",
			"deposit gopay", "depo gopay", "deposit linkaja", "depo linkaja",
			"deposit shopeepay", "deposit bank", "depo bank",
			"transfer bank", "deposit ewallet", "e-wallet slot",
			"qris slot", "qris judi", "via pulsa", "via dana",
			"pulsa telkomsel", "pulsa xl", "pulsa indosat", "pulsa three",
			"pulsa axis", "pulsa smartfren",
		},
		Description: "Metode pembayaran untuk judi online"},

	// URL Patterns for Gambling Sites
	{Category: DorkCategoryGambling, Name: "Gambling URL Patterns", Severity: DorkSeverityCritical, IsRegex: true, IsActive: true,
		Pattern:     `(?i)(slot|togel|casino|poker|judi|betting|taruhan|gacor|maxwin|scatter|jackpot|cuan|hoki|menang|raja|dewa|boss?|mpo|win|jp)\d*\.(com|net|org|io|site|online|xyz|club|vip|pro|bet|games|fun|live|today|info|biz|cc|me|co|id|asia)`,
		Description: "Pola URL situs judi"},

	// Gambling Redirect Scripts
	{Category: DorkCategoryGambling, Name: "Gambling Redirect Scripts", Severity: DorkSeverityCritical, IsRegex: true, IsActive: true,
		Pattern:     `(?i)(window\.location|location\.href|location\.replace|document\.location|top\.location)\s*=\s*['"](https?://)?[^'"]*?(slot|togel|casino|judi|poker|betting|gacor|maxwin|cuan)`,
		Description: "Script redirect ke situs judi"},

	// Hidden Gambling Links
	{Category: DorkCategoryGambling, Name: "Hidden Gambling Links", Severity: DorkSeverityHigh, IsRegex: true, IsActive: true,
		Pattern:     `(?i)<a[^>]*href\s*=\s*["'][^"']*?(slot|togel|casino|judi|poker|gacor|maxwin|raja|dewa|mpo)[^"']*?["'][^>]*(style\s*=\s*["'][^"']*?(display:\s*none|visibility:\s*hidden|opacity:\s*0|font-size:\s*0|height:\s*0|width:\s*0))`,
		Description: "Link judi tersembunyi"},

	// Gambling Meta Tags
	{Category: DorkCategoryGambling, Name: "Gambling Meta Tags", Severity: DorkSeverityHigh, IsRegex: true, IsActive: true,
		Pattern:     `(?i)<meta[^>]*(content|name|property)\s*=\s*["'][^"']*?(slot|togel|casino|judi|poker|gacor|maxwin|cuan|jackpot|bonus\s*member)[^"']*?["']`,
		Description: "Meta tag mengandung kata kunci judi"},

	// WhatsApp/Telegram Gambling Groups
	{Category: DorkCategoryGambling, Name: "Gambling Contact Links", Severity: DorkSeverityHigh, IsRegex: true, IsActive: true,
		Pattern:     `(?i)(wa\.me|api\.whatsapp\.com|t\.me|telegram\.me|chat\.whatsapp\.com)/[^\s"'<>]*`,
		Description: "Link kontak WA/Telegram untuk judi"},

	// Livechat Gambling
	{Category: DorkCategoryGambling, Name: "Livechat Gambling", Severity: DorkSeverityMedium, IsRegex: false, IsActive: true,
		Keywords: []string{
			"livechat 24 jam", "cs 24 jam", "customer service 24 jam",
			"hubungi cs", "chat admin", "live chat slot", "live chat togel",
			"bantuan member", "layanan 24 jam", "online 24 jam",
		},
		Description: "Kata kunci livechat situs judi"},

	// === DEFACEMENT PATTERNS ===
	{Category: DorkCategoryDefacement, Name: "Hacked By Signature", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(hacked\s*by|defaced\s*by|pwned\s*by|owned\s*by|greetz?\s*(to|from)|cyber\s*(army|team|crew)|h4ck[e3]d|d[e3]f[a4]c[e3]d)`,
		Description: "Detects common defacement signatures"},
	{Category: DorkCategoryDefacement, Name: "Defacement Messages", Severity: DorkSeverityCritical, IsRegex: false,
		Keywords: []string{
			"this site has been hacked", "website hacked", "your security sucks",
			"we are anonymous", "expect us", "we do not forgive", "we do not forget",
			"security breach", "data leaked", "hacked for", "defaced for",
		},
		Description: "Detects defacement message patterns"},
	{Category: DorkCategoryDefacement, Name: "Indonesian Defacer Groups", Severity: DorkSeverityCritical, IsRegex: false,
		Keywords: []string{
			"indonesian defacer", "garuda cyber", "indonesia cyber", "jatim cyber",
			"surabaya black hat", "jakarta cyber", "bali cyber", "hacker newbie",
			"indonesian hacker", "cyber indonesia", "defacer indonesia",
		},
		Description: "Detects Indonesian defacer group signatures"},
	{Category: DorkCategoryDefacement, Name: "Defacement Image Patterns", Severity: DorkSeverityHigh, IsRegex: true,
		Pattern:     `(?i)<img[^>]*src\s*=\s*["'][^"']*?(defaced|hacked|pwned|skull|anonymous)[^"']*?["']`,
		Description: "Detects defacement-related images"},
	{Category: DorkCategoryDefacement, Name: "Zone-H Mirror", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(zone-?h\.(com|org)|mirror|notified\s*by)`,
		Description: "Detects Zone-H defacement archive references"},

	// === WEBSHELL PATTERNS ===
	{Category: DorkCategoryShell, Name: "PHP Webshell Functions", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(eval\s*\(\s*\$_(GET|POST|REQUEST|COOKIE)|base64_decode\s*\(\s*\$_|shell_exec|passthru|system\s*\(\s*\$_|exec\s*\(\s*\$_|popen\s*\(|proc_open)`,
		Description: "Detects PHP webshell function patterns"},
	{Category: DorkCategoryShell, Name: "Common Webshell Names", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(c99|r57|b374k|wso|alfa|mini\s*shell|php\s*spy|chaos\s*shell|cmd\.php|shell\.php|backdoor\.php|upload\.php|filemanager\.php)`,
		Description: "Detects common webshell filenames"},
	{Category: DorkCategoryShell, Name: "Webshell Signatures", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(uname\s*-a|safe[_\s]*mode|server\s*ip|php\s*version|disable[_\s]*functions|document[_\s]*root|getcwd|chmod|mkdir|file[_\s]*get[_\s]*contents.*http)`,
		Description: "Detects webshell interface signatures"},
	{Category: DorkCategoryShell, Name: "Encoded Payloads", Severity: DorkSeverityHigh, IsRegex: true,
		Pattern:     `(?i)(eval\s*\(\s*gzinflate|eval\s*\(\s*gzuncompress|eval\s*\(\s*str_rot13|preg_replace\s*\([^)]*\/e)`,
		Description: "Detects encoded/obfuscated payloads"},

	// === MALWARE PATTERNS ===
	{Category: DorkCategoryMalware, Name: "Cryptocurrency Mining", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(coinhive|cryptonight|coin-?hive|minero|miner\.start|webminer|crypto-?loot|coinimp|jsecoin)`,
		Description: "Detects cryptocurrency mining scripts"},
	{Category: DorkCategoryMalware, Name: "Malicious Iframe", Severity: DorkSeverityHigh, IsRegex: true,
		Pattern:     `(?i)<iframe[^>]*style\s*=\s*["'][^"']*(display:\s*none|visibility:\s*hidden|width:\s*[01]px|height:\s*[01]px)`,
		Description: "Detects hidden/malicious iframes"},
	{Category: DorkCategoryMalware, Name: "Drive-by Download", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(document\.write\s*\(\s*unescape|\.exe["']|\.scr["']|\.bat["']|\.cmd["']|\.vbs["']|\.jar["'])`,
		Description: "Detects drive-by download attempts"},
	{Category: DorkCategoryMalware, Name: "Obfuscated JavaScript", Severity: DorkSeverityMedium, IsRegex: true,
		Pattern:     `(?i)(eval\s*\(\s*function\s*\(p,a,c,k,e,[dr]\)|String\.fromCharCode\s*\([^)]{100,}|\\x[0-9a-f]{2}){5,}`,
		Description: "Detects heavily obfuscated JavaScript"},
	{Category: DorkCategoryMalware, Name: "External Malicious Scripts", Severity: DorkSeverityHigh, IsRegex: true,
		Pattern:     `(?i)<script[^>]*src\s*=\s*["']https?://[^"']*(\.xyz|\.tk|\.ml|\.ga|\.cf|\.cc|\.su|\.ws|\.top|\.club|\.vip)[^"']*\.js["']`,
		Description: "Detects external scripts from suspicious TLDs"},

	// === PHISHING PATTERNS ===
	{Category: DorkCategoryPhishing, Name: "Login Form Suspicious", Severity: DorkSeverityHigh, IsRegex: true,
		Pattern:     `(?i)<form[^>]*action\s*=\s*["']https?://[^"']*(\.xyz|\.tk|\.ml|\.ga|\.cf|\.cc|\.su|\.ws|\.top|\.club|\.vip)[^"']*["']`,
		Description: "Detects login forms posting to suspicious domains"},
	{Category: DorkCategoryPhishing, Name: "Fake Bank Keywords", Severity: DorkSeverityCritical, IsRegex: false,
		Keywords: []string{
			"verifikasi akun", "konfirmasi data", "update data nasabah",
			"blokir rekening", "transaksi mencurigakan", "keamanan akun",
			"klik disini untuk verifikasi", "masukkan pin", "masukkan otp",
		},
		Description: "Detects fake banking phishing keywords"},
	{Category: DorkCategoryPhishing, Name: "Government Phishing", Severity: DorkSeverityCritical, IsRegex: false,
		Keywords: []string{
			"bantuan pemerintah palsu", "daftar bansos", "subsidi palsu",
			"klaim bantuan", "verifikasi nik", "daftar bpjs palsu",
		},
		Description: "Detects government assistance phishing"},

	// === SEO SPAM PATTERNS ===
	{Category: DorkCategorySEOSpam, Name: "Hidden Text SEO Spam", Severity: DorkSeverityMedium, IsRegex: true,
		Pattern:     `(?i)<(div|span|p)[^>]*style\s*=\s*["'][^"']*?(color:\s*(#fff|white|transparent)|font-size:\s*[01]px|display:\s*none)[^"']*["'][^>]*>[^<]*?(slot|togel|casino|judi|viagra|cialis|porn)`,
		Description: "Detects hidden SEO spam text"},
	{Category: DorkCategorySEOSpam, Name: "Keyword Stuffing", Severity: DorkSeverityMedium, IsRegex: true,
		Pattern:     `(?i)(slot|togel|judi|casino|poker){3,}`,
		Description: "Detects keyword stuffing"},
	{Category: DorkCategorySEOSpam, Name: "Spam Link Injection", Severity: DorkSeverityHigh, IsRegex: true,
		Pattern:     `(?i)<a[^>]*href\s*=\s*["'][^"']*?(buy|cheap|discount|pharmacy|pills|viagra|cialis|replica|outlet)[^"']*["']`,
		Description: "Detects spam link injections"},
	{Category: DorkCategorySEOSpam, Name: "Japanese SEO Spam", Severity: DorkSeverityHigh, IsRegex: true,
		Pattern:     `[\x{3040}-\x{309F}\x{30A0}-\x{30FF}]{10,}`,
		Description: "Detects Japanese keyword spam injection"},

	// === BACKDOOR PATTERNS ===
	{Category: DorkCategoryBackdoor, Name: "File Upload Backdoor", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(move_uploaded_file|copy\s*\([^)]*\$_(FILES|GET|POST)|file_put_contents\s*\([^)]*\$_)`,
		Description: "Detects file upload backdoors"},
	{Category: DorkCategoryBackdoor, Name: "Database Backdoor", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(mysql_query\s*\([^)]*\$_|mysqli_query\s*\([^)]*\$_|->query\s*\([^)]*\$_)`,
		Description: "Detects SQL injection backdoors"},
	{Category: DorkCategoryBackdoor, Name: "Remote Include", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(include|require|include_once|require_once)\s*\(\s*["']?(https?://|\$_(GET|POST|REQUEST))`,
		Description: "Detects remote file inclusion"},

	// === INJECTION PATTERNS ===
	{Category: DorkCategoryInjection, Name: "Script Injection", Severity: DorkSeverityHigh, IsRegex: true,
		Pattern:     `(?i)<script[^>]*>[^<]*?(document\.cookie|localStorage|sessionStorage|\.innerHTML\s*=)`,
		Description: "Detects potential XSS script injections"},
	{Category: DorkCategoryInjection, Name: "SQL Injection Traces", Severity: DorkSeverityCritical, IsRegex: true,
		Pattern:     `(?i)(UNION\s+SELECT|DROP\s+TABLE|INSERT\s+INTO|DELETE\s+FROM|UPDATE\s+.*SET|1=1|OR\s+1=1)`,
		Description: "Detects SQL injection traces"},
	{Category: DorkCategoryInjection, Name: "HTML Injection", Severity: DorkSeverityMedium, IsRegex: true,
		Pattern:     `(?i)(<script|<iframe|<object|<embed|<form|javascript:|data:text/html|vbscript:)`,
		Description: "Detects HTML/JavaScript injection"},
}

// DorkPatternFilter for filtering patterns
type DorkPatternFilter struct {
	Category  DorkCategory `json:"category"`
	Severity  DorkSeverity `json:"severity"`
	IsActive  *bool        `json:"is_active"`
	IsDefault *bool        `json:"is_default"`
}

// DorkDetectionFilter for filtering detections
type DorkDetectionFilter struct {
	WebsiteID       int64        `json:"website_id"`
	ScanResultID    int64        `json:"scan_result_id"`
	Category        DorkCategory `json:"category"`
	Severity        DorkSeverity `json:"severity"`
	IsResolved      *bool        `json:"is_resolved"`
	IsFalsePositive *bool        `json:"is_false_positive"`
	Limit           int          `json:"limit"`
	Offset          int          `json:"offset"`
}

// WebsiteDorkSettings represents per-website dork scan settings
type WebsiteDorkSettings struct {
	ID                int64          `db:"id" json:"id"`
	WebsiteID         int64          `db:"website_id" json:"website_id"`
	IsEnabled         bool           `db:"is_enabled" json:"is_enabled"`
	ScanFrequency     string         `db:"scan_frequency" json:"scan_frequency"` // hourly, daily, weekly, manual
	ScanDepth         int            `db:"scan_depth" json:"scan_depth"`
	MaxPages          int            `db:"max_pages" json:"max_pages"`
	CategoriesEnabled []DorkCategory `json:"categories_enabled"`
	ExcludedPaths     []string       `json:"excluded_paths"`
	LastScanAt        *time.Time     `db:"last_scan_at" json:"last_scan_at,omitempty"`
	NextScanAt        *time.Time     `db:"next_scan_at" json:"next_scan_at,omitempty"`
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at" json:"updated_at"`
}

// DorkDetectionStats represents detection statistics for a website
type DorkDetectionStats struct {
	WebsiteID       int64                     `json:"website_id"`
	TotalDetections int                       `json:"total_detections"`
	UnresolvedCount int                       `json:"unresolved_count"`
	ByCategory      map[DorkCategory]int      `json:"by_category"`
	BySeverity      map[DorkSeverity]int      `json:"by_severity"`
	LastScanAt      *time.Time                `json:"last_scan_at,omitempty"`
}

// DorkOverallStats represents overall dork monitoring statistics
type DorkOverallStats struct {
	TotalScans       int                       `json:"total_scans"`
	TotalDetections  int                       `json:"total_detections"`
	UnresolvedCount  int                       `json:"unresolved_count"`
	WebsitesAffected int                       `json:"websites_affected"`
	ByCategory       map[DorkCategory]int      `json:"by_category"`
	BySeverity       map[DorkSeverity]int      `json:"by_severity"`
}

// Domain suffixes commonly used by gambling sites (from BrigsLabs blocklist)
var GamblingSiteSuffixes = []string{
	".com", ".net", ".org", ".io", ".site", ".online", ".xyz", ".club",
	".vip", ".pro", ".bet", ".games", ".fun", ".live", ".today", ".info",
	".biz", ".cc", ".me", ".co", ".asia", ".sbs", ".lol", ".cyou",
	".shop", ".store", ".top", ".win", ".casino", ".poker", ".slot",
}

// Google Dork queries for external search (reference - based on arxiv research)
var GoogleDorkQueries = map[DorkCategory][]string{
	DorkCategoryGambling: {
		// Core keywords with lowest false positive (from arxiv research)
		`site:*.go.id "togel"`,
		`site:*.go.id "toto"`,
		`site:*.go.id "slot"`,
		`site:*.go.id "gacor"`,
		`site:*.go.id "bandar"`,
		`site:*.go.id "maxwin"`,
		`site:*.go.id "zeus"`,
		// Combined queries
		`site:*.go.id "slot gacor"`,
		`site:*.go.id "togel online"`,
		`site:*.go.id "judi online"`,
		`site:*.go.id "casino online"`,
		`site:*.go.id "sbobet"`,
		// URL patterns
		`site:*.go.id inurl:slot`,
		`site:*.go.id inurl:togel`,
		`site:*.go.id inurl:gacor`,
		`site:*.go.id inurl:judol`,
		// Title patterns
		`site:*.go.id intitle:"slot gacor"`,
		`site:*.go.id intitle:"togel online"`,
		`site:*.go.id intitle:"judi online"`,
		// Folder patterns (from CSIRT guide)
		`site:*.go.id inurl:/slot-gacor/`,
		`site:*.go.id inurl:/judi-online/`,
		// Academic sites (.ac.id)
		`site:*.ac.id "slot gacor"`,
		`site:*.ac.id "togel"`,
		`site:*.ac.id "maxwin"`,
	},
	DorkCategoryDefacement: {
		`site:*.go.id "hacked by"`,
		`site:*.go.id "defaced by"`,
		`site:*.go.id "owned by"`,
		`site:*.go.id "pwned by"`,
		`site:*.go.id intitle:"hacked"`,
		`site:*.go.id intitle:"defaced"`,
		`site:*.go.id "cyber army"`,
		`site:*.go.id "indonesian defacer"`,
		`site:*.go.id "garuda cyber"`,
	},
	DorkCategoryShell: {
		`site:*.go.id inurl:shell.php`,
		`site:*.go.id inurl:c99.php`,
		`site:*.go.id inurl:r57.php`,
		`site:*.go.id inurl:b374k`,
		`site:*.go.id inurl:wso.php`,
		`site:*.go.id filetype:php "eval(base64_decode"`,
		`site:*.go.id filetype:php "shell_exec"`,
		`site:*.go.id filetype:php "passthru"`,
	},
	DorkCategoryMalware: {
		`site:*.go.id "coinhive"`,
		`site:*.go.id "cryptoloot"`,
		`site:*.go.id inurl:.exe`,
		`site:*.go.id filetype:exe`,
	},
	DorkCategoryPhishing: {
		`site:*.go.id "verifikasi akun"`,
		`site:*.go.id "konfirmasi data"`,
		`site:*.go.id "update data nasabah"`,
		`site:*.go.id intitle:"login" -inurl:login`,
	},
	DorkCategorySEOSpam: {
		`site:*.go.id "buy viagra"`,
		`site:*.go.id "cheap pills"`,
		`site:*.go.id "pharmacy online"`,
	},
}

// FalsePositiveKeywords - Keywords that may cause false positives (from arxiv research)
// Note: "bet" keyword has 60 false positives due to matching "between", "beta", etc.
var FalsePositiveKeywords = []string{
	"bet",      // matches "between", "beta", "better"
	"win",      // matches "window", "windows", "winning" (legitimate)
	"game",     // matches legitimate gaming content
	"play",     // matches legitimate play content
	"bonus",    // matches legitimate bonus programs
	"deposit",  // matches legitimate bank deposits
	"transfer", // matches legitimate transfers
}
