-- 回退 000001_dictionary.up.sql：索引与约束随表一起删除。
-- 前置：000002 已回退（user_words 外键引用本表）。
DROP TABLE dictionary_entries;
