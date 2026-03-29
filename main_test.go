package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "test-db"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "root"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "tracker_test"
	}
	dsn := fmt.Sprintf("user=%s dbname=%s password=%s sslmode=disable host=%s port=%s", user, dbname, password, host, port)

	var err error
	testDB, err = sql.Open("postgres", dsn)
	if err != nil {
		fmt.Printf("Failed to connect to test DB: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	_, err = testDB.Exec("TRUNCATE users, review_periods, reviews RESTART IDENTITY CASCADE")
	if err != nil {
		fmt.Printf("Failed to truncate tables: %v\n", err)
		os.Exit(1)
	}

	_, err = testDB.Exec(`
		INSERT INTO users (name, role) VALUES
			('Admin', 'admin'),
			('Manager', 'manager'),
			('Employee', 'employee');
		INSERT INTO review_periods (name, start_date, end_date) VALUES
			('Test Period', '2025-01-01', '2025-03-31');
		INSERT INTO reviews (employee_id, reviewer_id, period_id, soft_skills_score, hard_skills_score, comment, status, created_at, updated_at)
		VALUES (3, 2, 1, 4, 5, 'Good work', 'published', NOW(), NOW());
	`)
	if err != nil {
		fmt.Printf("Failed to seed test data: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}

// Тест 1: валидация оценок
func TestReviewInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   ReviewInput
		wantErr bool
	}{
		{"valid", ReviewInput{SoftSkillsScore: 3, HardSkillsScore: 4, Comment: "good"}, false},
		{"soft too low", ReviewInput{SoftSkillsScore: 0, HardSkillsScore: 4, Comment: "good"}, true},
		{"soft too high", ReviewInput{SoftSkillsScore: 6, HardSkillsScore: 4, Comment: "good"}, true},
		{"hard too low", ReviewInput{SoftSkillsScore: 3, HardSkillsScore: 0, Comment: "good"}, true},
		{"hard too high", ReviewInput{SoftSkillsScore: 3, HardSkillsScore: 6, Comment: "good"}, true},
		{"empty comment", ReviewInput{SoftSkillsScore: 3, HardSkillsScore: 4, Comment: ""}, true},
		{"comment only spaces", ReviewInput{SoftSkillsScore: 3, HardSkillsScore: 4, Comment: "   "}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Тест 2: после публикации отзыв нельзя редактировать
func TestCannotEditPublishedReview(t *testing.T) {
	reqBody := ReviewInput{
		EmployeeID:      3,
		ReviewerID:      2,
		PeriodID:        1,
		SoftSkillsScore: 3,
		HardSkillsScore: 3,
		Comment:         "Trying to edit",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/api/reviews/1", bytes.NewReader(body))
	req.Header.Set("X-User-Id", "1")
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "1"})

	rr := httptest.NewRecorder()
	handler := updateReview(testDB)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 Forbidden, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// Тест 3: employee видит только свои опубликованные отзывы
func TestEmployeeSeesOnlyOwnPublishedReviews(t *testing.T) {
	_, err := testDB.Exec(`
		INSERT INTO reviews (employee_id, reviewer_id, period_id, soft_skills_score, hard_skills_score, comment, status, created_at, updated_at)
		VALUES (3, 2, 1, 3, 4, 'Draft review', 'draft', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("Failed to insert draft review: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/reviews", nil)
	req.Header.Set("X-User-Id", "3")

	rr := httptest.NewRecorder()
	handler := apiReviewsList(testDB)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}

	var reviews []Review
	err = json.NewDecoder(rr.Body).Decode(&reviews)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(reviews) != 1 {
		t.Errorf("Expected 1 review, got %d", len(reviews))
	}
	if len(reviews) > 0 && reviews[0].Status != "published" {
		t.Errorf("Expected status 'published', got '%s'", reviews[0].Status)
	}
}
