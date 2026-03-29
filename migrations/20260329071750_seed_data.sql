-- +goose Up
INSERT INTO users (name, role) VALUES
    ('Анна Администратор', 'admin'),
    ('Борис Менеджер', 'manager'),
    ('Виктория Сотрудник', 'employee');

INSERT INTO review_periods (name, start_date, end_date) VALUES
    ('Q1 2025', '2025-01-01', '2025-03-31'),
    ('Q2 2025', '2025-04-01', '2025-06-30');

INSERT INTO reviews (employee_id, reviewer_id, period_id, soft_skills_score, hard_skills_score, comment, status) VALUES
    (3, 2, 1, 4, 5, 'Отличная работа в первом квартале', 'published'),
    (3, 2, 2, 3, 4, 'Хорошо, но есть потенциал для роста', 'draft');