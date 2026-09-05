-- 迁移 000001：词典域表（docs/backend/structure.md §2）。
-- dictionary_entries 是全局共享的客观词典数据（D006），字段语义以
-- docs/dictionary/data-model.md 为准；用户学习数据在 000002 的 user_words。

CREATE TABLE dictionary_entries (
    id              uuid PRIMARY KEY,
    headword        text NOT NULL,
    lemma           text NOT NULL,
    phonetic_uk     text NOT NULL DEFAULT '',
    phonetic_us     text NOT NULL DEFAULT '',
    raw_meanings    jsonb NOT NULL DEFAULT '[]',
    review_meanings jsonb,
    exchange        jsonb NOT NULL DEFAULT '{}',
    frequency       jsonb NOT NULL DEFAULT '{}',
    tags            jsonb NOT NULL DEFAULT '[]',
    source          text NOT NULL DEFAULT '',
    source_version  text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),

    -- 释义列的 JSON 形态即 data-model.md §4 的义项数组（review_meanings
    -- 可空表示未设置），对象或标量都是坏数据。
    CONSTRAINT dictionary_entries_raw_meanings_check
        CHECK (jsonb_typeof(raw_meanings) = 'array'),
    CONSTRAINT dictionary_entries_review_meanings_check
        CHECK (review_meanings IS NULL OR jsonb_typeof(review_meanings) = 'array'),
    CONSTRAINT dictionary_entries_exchange_check
        CHECK (jsonb_typeof(exchange) = 'object'),
    CONSTRAINT dictionary_entries_frequency_check
        CHECK (jsonb_typeof(frequency) = 'object'),
    CONSTRAINT dictionary_entries_tags_check
        CHECK (jsonb_typeof(tags) = 'array')
);

-- headword 唯一：词条唯一键，Lookup 未命中创建词条与 ECDICT 导入的
-- 并发 / 幂等重跑都依赖它（structure.md §5.3）。
CREATE UNIQUE INDEX dictionary_entries_headword_key
    ON dictionary_entries (headword);

-- lemma 查询：DictionaryService 的 lemma 候选查询按 lemma 检索词条
-- （structure.md §4.1）。
CREATE INDEX dictionary_entries_lemma_idx
    ON dictionary_entries (lemma);
