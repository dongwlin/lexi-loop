-- 回退 000002_review.up.sql：索引与约束随表一起删除。
-- 先删引用方，再删被引用方。
DROP TABLE review_items;
DROP TABLE review_sessions;
DROP TABLE user_words;
