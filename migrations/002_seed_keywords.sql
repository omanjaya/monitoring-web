-- Seed data for gambling keywords
INSERT INTO keywords (keyword, category, is_regex, weight, is_active) VALUES
('slot gacor', 'gambling', FALSE, 10, TRUE),
('slot online', 'gambling', FALSE, 10, TRUE),
('judi online', 'gambling', FALSE, 10, TRUE),
('togel', 'gambling', FALSE, 10, TRUE),
('casino', 'gambling', FALSE, 8, TRUE),
('poker online', 'gambling', FALSE, 8, TRUE),
('pragmatic', 'gambling', FALSE, 9, TRUE),
('joker123', 'gambling', FALSE, 10, TRUE),
('sbobet', 'gambling', FALSE, 10, TRUE),
('maxwin', 'gambling', FALSE, 9, TRUE),
('scatter', 'gambling', FALSE, 7, TRUE),
('jackpot', 'gambling', FALSE, 6, TRUE),
('rtp slot', 'gambling', FALSE, 9, TRUE),
('bocoran slot', 'gambling', FALSE, 10, TRUE),
('demo slot', 'gambling', FALSE, 7, TRUE),
('bandar togel', 'gambling', FALSE, 10, TRUE),
('live casino', 'gambling', FALSE, 8, TRUE),
('deposit pulsa', 'gambling', FALSE, 7, TRUE),
('slot88', 'gambling', FALSE, 10, TRUE),
('slot777', 'gambling', FALSE, 10, TRUE),
('gacor hari ini', 'gambling', FALSE, 10, TRUE),
('pg soft', 'gambling', FALSE, 8, TRUE),
('habanero', 'gambling', FALSE, 8, TRUE),
('spadegaming', 'gambling', FALSE, 8, TRUE),
('bola online', 'gambling', FALSE, 8, TRUE),
('taruhan bola', 'gambling', FALSE, 9, TRUE),
('agen slot', 'gambling', FALSE, 10, TRUE),
('daftar slot', 'gambling', FALSE, 9, TRUE),
('link alternatif', 'gambling', FALSE, 6, TRUE),
('bonus new member', 'gambling', FALSE, 7, TRUE)
ON DUPLICATE KEY UPDATE keyword = VALUES(keyword);

-- Seed data for defacement keywords
INSERT INTO keywords (keyword, category, is_regex, weight, is_active) VALUES
('hacked by', 'defacement', FALSE, 10, TRUE),
('defaced by', 'defacement', FALSE, 10, TRUE),
('owned by', 'defacement', FALSE, 9, TRUE),
('greetz to', 'defacement', FALSE, 10, TRUE),
('cyber army', 'defacement', FALSE, 9, TRUE),
('indonesian hacker', 'defacement', FALSE, 8, TRUE),
('security breach', 'defacement', FALSE, 7, TRUE),
('anonymous', 'defacement', FALSE, 6, TRUE),
('1337 hax0r', 'defacement', FALSE, 10, TRUE),
('pwned', 'defacement', FALSE, 8, TRUE)
ON DUPLICATE KEY UPDATE keyword = VALUES(keyword);

-- Seed data for porn keywords
INSERT INTO keywords (keyword, category, is_regex, weight, is_active) VALUES
('bokep', 'porn', FALSE, 10, TRUE),
('porn', 'porn', FALSE, 10, TRUE),
('xxx', 'porn', FALSE, 9, TRUE),
('sex video', 'porn', FALSE, 10, TRUE),
('adult content', 'porn', FALSE, 8, TRUE)
ON DUPLICATE KEY UPDATE keyword = VALUES(keyword);
