-- 迁移 000002：复习域表（docs/backend/structure.md §2）。
-- user_words 学习字段以 docs/dictionary/data-model.md §3 为准
-- （词典数据 ≠ 学习数据 D006，软删除 D004，weight / mastery 不落库 D011）；
-- review_sessions / review_items 以 docs/review/data-model.md 为准。

CREATE TABLE user_words (
    id                    uuid PRIMARY KEY,
    dictionary_entry_id   uuid NOT NULL,
    encounter_count       integer NOT NULL DEFAULT 0,
    review_count          integer NOT NULL DEFAULT 0,
    remember_count        integer NOT NULL DEFAULT 0,
    forget_count          integer NOT NULL DEFAULT 0,
    current_streak        integer NOT NULL DEFAULT 0,
    last_reviewed_at      timestamptz,
    custom_review_meaning jsonb,
    deleted_at            timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT user_words_dictionary_entry_id_fkey
        FOREIGN KEY (dictionary_entry_id) REFERENCES dictionary_entries (id),
    -- MVP 单用户：一个词条至多一行学习状态（多用户时改为
    -- UNIQUE(user_id, dictionary_entry_id)，见 architecture/data-model.md §3）。
    CONSTRAINT user_words_dictionary_entry_id_key
        UNIQUE (dictionary_entry_id),
    -- 词级累计不变量（dictionary/data-model.md §3）。
    CONSTRAINT user_words_review_counts_check
        CHECK (review_count = remember_count + forget_count),
    -- custom_review_meaning 与释义同构：NULL 表示未自定义，否则为义项数组。
    CONSTRAINT user_words_custom_review_meaning_check
        CHECK (custom_review_meaning IS NULL OR jsonb_typeof(custom_review_meaning) = 'array')
);

CREATE TABLE review_sessions (
    id               uuid PRIMARY KEY,
    requested_count  integer NOT NULL,
    total_count      integer NOT NULL,
    remembered_count integer NOT NULL DEFAULT 0,
    forgotten_count  integer NOT NULL DEFAULT 0,
    status           text NOT NULL DEFAULT 'active',
    started_at       timestamptz NOT NULL DEFAULT now(),
    completed_at     timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),

    -- status 取值（review/data-model.md §4）。
    CONSTRAINT review_sessions_status_check
        CHECK (status IN ('active', 'completed', 'abandoned')),
    -- D009：total = min(count, available)；Domain 校验后由数据库兜底。
    CONSTRAINT review_sessions_counts_check
        CHECK (requested_count >= 1 AND total_count >= 1 AND total_count <= requested_count),
    -- 汇总只在完成时回写，且不超过本轮总题数。
    CONSTRAINT review_sessions_totals_check
        CHECK (remembered_count + forgotten_count <= total_count)
);

-- D010：全库至多一个 active session 的部分唯一索引，作为并发创建流程
-- （Repo 封装的 advisory transaction lock）的最终防线。
CREATE UNIQUE INDEX review_sessions_single_active_key
    ON review_sessions (status)
    WHERE status = 'active';

CREATE TABLE review_items (
    id           uuid PRIMARY KEY,
    session_id   uuid NOT NULL,
    user_word_id uuid NOT NULL,
    position     integer NOT NULL,
    result       text NOT NULL DEFAULT 'pending',
    created_at   timestamptz NOT NULL DEFAULT now(),
    reviewed_at  timestamptz,

    CONSTRAINT review_items_session_id_fkey
        FOREIGN KEY (session_id) REFERENCES review_sessions (id),
    CONSTRAINT review_items_user_word_id_fkey
        FOREIGN KEY (user_word_id) REFERENCES user_words (id),
    -- 同一 session 内 position 唯一、同一 user_word 至多出现一次
    -- （review/data-model.md §2）。
    CONSTRAINT review_items_session_position_key
        UNIQUE (session_id, position),
    CONSTRAINT review_items_session_user_word_key
        UNIQUE (session_id, user_word_id),
    CONSTRAINT review_items_result_check
        CHECK (result IN ('pending', 'remembered', 'forgotten')),
    CONSTRAINT review_items_position_check
        CHECK (position >= 0)
);
