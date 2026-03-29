-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL
);

CREATE TABLE review_periods (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL
);

CREATE TABLE reviews (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_id BIGINT NOT NULL REFERENCES review_periods(id) ON DELETE CASCADE,
    soft_skills_score INT,
    hard_skills_score INT,
    comment TEXT,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT chk_scores_range CHECK (
        soft_skills_score BETWEEN 1 AND 5 AND 
        hard_skills_score BETWEEN 1 AND 5
    )
);

CREATE INDEX idx_reviews_employee_period ON reviews(employee_id, period_id);
CREATE INDEX idx_reviews_reviewer_period ON reviews(reviewer_id, period_id);
CREATE INDEX idx_reviews_status_created ON reviews(status, created_at);

-- +goose Down
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS review_periods;
DROP TABLE IF EXISTS users;