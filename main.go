package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type User struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Period struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type Review struct {
	ID              int64     `json:"id"`
	EmployeeID      int64     `json:"employee_id"`
	EmployeeName    string    `json:"employee_name"`
	ReviewerID      int64     `json:"reviewer_id"`
	ReviewerName    string    `json:"reviewer_name"`
	PeriodID        int64     `json:"period_id"`
	PeriodName      string    `json:"period_name"`
	PeriodStart     string    `json:"period_start"`
	PeriodEnd       string    `json:"period_end"`
	SoftSkillsScore int       `json:"soft_skills_score"`
	HardSkillsScore int       `json:"hard_skills_score"`
	Comment         string    `json:"comment"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ReviewInput struct {
	EmployeeID      int64  `json:"employee_id"`
	ReviewerID      int64  `json:"reviewer_id"`
	PeriodID        int64  `json:"period_id"`
	SoftSkillsScore int    `json:"soft_skills_score"`
	HardSkillsScore int    `json:"hard_skills_score"`
	Comment         string `json:"comment"`
}

func (r ReviewInput) Validate() error {
	if r.SoftSkillsScore < 1 || r.SoftSkillsScore > 5 {
		return errors.New("soft_skills_score must be between 1 and 5")
	}
	if r.HardSkillsScore < 1 || r.HardSkillsScore > 5 {
		return errors.New("hard_skills_score must be between 1 and 5")
	}
	if strings.TrimSpace(r.Comment) == "" {
		return errors.New("comment is required")
	}
	return nil
}

func getCurrentUser(r *http.Request, db *sql.DB) (*User, error) {
	userIDStr := r.Header.Get("X-User-Id")
	if userIDStr == "" {
		cookie, err := r.Cookie("user_id")
		if err == nil && cookie != nil {
			userIDStr = cookie.Value
		}
	}
	if userIDStr == "" {
		return nil, errors.New("missing user identification")
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	var user User
	err = db.QueryRow("SELECT id, name, role FROM users WHERE id = $1", userID).
		Scan(&user.Id, &user.Name, &user.Role)
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func checkAdminManager(user *User) bool {
	return user.Role == "admin" || user.Role == "manager"
}

func getPeriods(db *sql.DB) ([]Period, error) {
	rows, err := db.Query("SELECT id, name, start_date, end_date FROM review_periods ORDER BY start_date DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var periods []Period
	for rows.Next() {
		var p Period
		var start, end time.Time
		if err := rows.Scan(&p.ID, &p.Name, &start, &end); err != nil {
			return nil, err
		}
		p.StartDate = start.Format("2006-01-02")
		p.EndDate = end.Format("2006-01-02")
		periods = append(periods, p)
	}
	return periods, rows.Err()
}

func getEmployees(db *sql.DB) ([]User, error) {
	rows, err := db.Query("SELECT id, name, role FROM users ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.Id, &u.Name, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func index(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "index", nil)
}

func usersPage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("templates/users.html", "templates/header.html", "templates/footer.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rows, err := db.Query("SELECT id, name, role FROM users")
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var users []User
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.Id, &u.Name, &u.Role); err != nil {
				http.Error(w, "Scan error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			users = append(users, u)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, "Rows error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "users", users)
	}
}

func reviewsPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/reviews.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "reviews", nil)
}

func myReviewsPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/my_reviews.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "my_reviews", nil)
}

func reviewDetailPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/review.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "review", nil)
}

func newReviewForm(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		employees, err := getEmployees(db)
		if err != nil {
			http.Error(w, "Error loading employees: "+err.Error(), http.StatusInternalServerError)
			return
		}
		periods, err := getPeriods(db)
		if err != nil {
			http.Error(w, "Error loading periods: "+err.Error(), http.StatusInternalServerError)
			return
		}
		data := struct {
			Employees []User
			Periods   []Period
		}{employees, periods}
		tmpl, err := template.ParseFiles("templates/review_form.html", "templates/header.html", "templates/footer.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "review_form", data)
	}
}

func editReviewForm(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "Invalid review ID", http.StatusBadRequest)
			return
		}
		var review Review
		var softScore, hardScore sql.NullInt64
		var comment sql.NullString
		err = db.QueryRow(`
			SELECT id, employee_id, reviewer_id, period_id, soft_skills_score, hard_skills_score, comment, status
			FROM reviews WHERE id = $1
		`, id).Scan(&review.ID, &review.EmployeeID, &review.ReviewerID, &review.PeriodID,
			&softScore, &hardScore, &comment, &review.Status)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Review not found", http.StatusNotFound)
			} else {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if review.Status != "draft" {
			http.Error(w, "Cannot edit published review", http.StatusForbidden)
			return
		}
		user, err := getCurrentUser(r, db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if user.Role == "manager" && review.ReviewerID != user.Id {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if softScore.Valid {
			review.SoftSkillsScore = int(softScore.Int64)
		}
		if hardScore.Valid {
			review.HardSkillsScore = int(hardScore.Int64)
		}
		if comment.Valid {
			review.Comment = comment.String
		}
		employees, err := getEmployees(db)
		if err != nil {
			http.Error(w, "Error loading employees: "+err.Error(), http.StatusInternalServerError)
			return
		}
		periods, err := getPeriods(db)
		if err != nil {
			http.Error(w, "Error loading periods: "+err.Error(), http.StatusInternalServerError)
			return
		}
		data := struct {
			Review    Review
			Employees []User
			Periods   []Period
		}{review, employees, periods}
		tmpl, err := template.ParseFiles("templates/review_form.html", "templates/header.html", "templates/footer.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "review_form", data)
	}
}

func periodsListPage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		periods, err := getPeriods(db)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl, err := template.ParseFiles("templates/periods.html", "templates/header.html", "templates/footer.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "periods", periods)
	}
}

func newPeriodForm(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		tmpl, err := template.ParseFiles("templates/period_form.html", "templates/header.html", "templates/footer.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "period_form", nil)
	}
}

func editPeriodForm(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "Invalid period ID", http.StatusBadRequest)
			return
		}
		var period Period
		var start, end time.Time
		err = db.QueryRow("SELECT id, name, start_date, end_date FROM review_periods WHERE id=$1", id).
			Scan(&period.ID, &period.Name, &start, &end)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Period not found", http.StatusNotFound)
			} else {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		period.StartDate = start.Format("2006-01-02")
		period.EndDate = end.Format("2006-01-02")
		tmpl, err := template.ParseFiles("templates/period_form.html", "templates/header.html", "templates/footer.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "period_form", period)
	}
}

func apiMe(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func apiEmployees(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		employees, err := getEmployees(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(employees)
	}
}

func apiPeriods(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		periods, err := getPeriods(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(periods)
	}
}

func apiReviewsList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		periodID := r.URL.Query().Get("period_id")
		employeeID := r.URL.Query().Get("employee_id")
		query := `
			SELECT 
				r.id, r.employee_id, emp.name AS employee_name,
				r.reviewer_id, rev.name AS reviewer_name,
				r.period_id, rp.name AS period_name,
				r.soft_skills_score, r.hard_skills_score,
				r.comment, r.status, r.created_at, r.updated_at
			FROM reviews r
			JOIN users emp ON r.employee_id = emp.id
			JOIN users rev ON r.reviewer_id = rev.id
			JOIN review_periods rp ON r.period_id = rp.id
			WHERE 1=1
		`
		args := []interface{}{}
		argCounter := 1
		switch user.Role {
		case "admin":
		case "manager":
			query += fmt.Sprintf(" AND r.reviewer_id = $%d", argCounter)
			args = append(args, user.Id)
			argCounter++
		case "employee":
			query += fmt.Sprintf(" AND r.employee_id = $%d AND r.status = 'published'", argCounter)
			args = append(args, user.Id)
			argCounter++
		default:
			http.Error(w, "Unknown role", http.StatusForbidden)
			return
		}
		if periodID != "" {
			query += fmt.Sprintf(" AND r.period_id = $%d", argCounter)
			args = append(args, periodID)
			argCounter++
		}
		if employeeID != "" && (user.Role == "admin" || user.Role == "manager") {
			query += fmt.Sprintf(" AND r.employee_id = $%d", argCounter)
			args = append(args, employeeID)
			argCounter++
		}
		query += " ORDER BY r.created_at DESC"
		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var reviews []Review
		for rows.Next() {
			var r Review
			var softScore, hardScore sql.NullInt64
			var comment sql.NullString
			err := rows.Scan(&r.ID, &r.EmployeeID, &r.EmployeeName, &r.ReviewerID, &r.ReviewerName,
				&r.PeriodID, &r.PeriodName, &softScore, &hardScore, &comment, &r.Status, &r.CreatedAt, &r.UpdatedAt)
			if err != nil {
				http.Error(w, "Scan error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			if softScore.Valid {
				r.SoftSkillsScore = int(softScore.Int64)
			}
			if hardScore.Valid {
				r.HardSkillsScore = int(hardScore.Int64)
			}
			if comment.Valid {
				r.Comment = comment.String
			}
			reviews = append(reviews, r)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, "Rows error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reviews)
	}
}

func apiReviewDetail(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		user, err := getCurrentUser(r, db)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var review Review
		var softScore, hardScore sql.NullInt64
		var comment sql.NullString
		var startDate, endDate time.Time
		err = db.QueryRow(`
			SELECT 
				r.id, r.employee_id, emp.name AS employee_name,
				r.reviewer_id, rev.name AS reviewer_name,
				r.period_id, rp.name AS period_name,
				rp.start_date, rp.end_date,
				r.soft_skills_score, r.hard_skills_score,
				r.comment, r.status, r.created_at, r.updated_at
			FROM reviews r
			JOIN users emp ON r.employee_id = emp.id
			JOIN users rev ON r.reviewer_id = rev.id
			JOIN review_periods rp ON r.period_id = rp.id
			WHERE r.id = $1
		`, id).Scan(&review.ID, &review.EmployeeID, &review.EmployeeName, &review.ReviewerID, &review.ReviewerName,
			&review.PeriodID, &review.PeriodName, &startDate, &endDate, &softScore, &hardScore, &comment,
			&review.Status, &review.CreatedAt, &review.UpdatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Review not found", http.StatusNotFound)
			} else {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		switch user.Role {
		case "admin":
		case "manager":
			if review.ReviewerID != user.Id {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		case "employee":
			if review.EmployeeID != user.Id || review.Status != "published" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		default:
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if softScore.Valid {
			review.SoftSkillsScore = int(softScore.Int64)
		}
		if hardScore.Valid {
			review.HardSkillsScore = int(hardScore.Int64)
		}
		if comment.Valid {
			review.Comment = comment.String
		}
		review.PeriodStart = startDate.Format("2006-01-02")
		review.PeriodEnd = endDate.Format("2006-01-02")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(review)
	}
}

func createReview(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil || !checkAdminManager(user) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		var input ReviewInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if err := input.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", input.EmployeeID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "Employee not found", http.StatusBadRequest)
			return
		}
		var reviewerRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = $1", input.ReviewerID).Scan(&reviewerRole)
		if err != nil || (reviewerRole != "admin" && reviewerRole != "manager") {
			http.Error(w, "Reviewer must be admin or manager", http.StatusBadRequest)
			return
		}
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM review_periods WHERE id = $1)", input.PeriodID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "Period not found", http.StatusBadRequest)
			return
		}
		var id int64
		err = db.QueryRow(`
			INSERT INTO reviews (employee_id, reviewer_id, period_id, soft_skills_score, hard_skills_score, comment, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'draft', NOW(), NOW())
			RETURNING id
		`, input.EmployeeID, input.ReviewerID, input.PeriodID, input.SoftSkillsScore, input.HardSkillsScore, input.Comment).Scan(&id)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{"id": id})
	}
}

func updateReview(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil || !checkAdminManager(user) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "Invalid review ID", http.StatusBadRequest)
			return
		}
		var status string
		var reviewerID int64
		err = db.QueryRow("SELECT status, reviewer_id FROM reviews WHERE id = $1", id).Scan(&status, &reviewerID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Review not found", http.StatusNotFound)
			} else {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if status != "draft" {
			http.Error(w, "Cannot edit published review", http.StatusForbidden)
			return
		}
		if user.Role == "manager" && reviewerID != user.Id {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		var input ReviewInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if err := input.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, err = db.Exec(`
			UPDATE reviews
			SET soft_skills_score = $1, hard_skills_score = $2, comment = $3, updated_at = NOW()
			WHERE id = $4
		`, input.SoftSkillsScore, input.HardSkillsScore, input.Comment, id)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func publishReview(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil || !checkAdminManager(user) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "Invalid review ID", http.StatusBadRequest)
			return
		}
		var status string
		var reviewerID int64
		err = db.QueryRow("SELECT status, reviewer_id FROM reviews WHERE id = $1", id).Scan(&status, &reviewerID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Review not found", http.StatusNotFound)
			} else {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if status != "draft" {
			http.Error(w, "Review already published", http.StatusForbidden)
			return
		}
		if user.Role == "manager" && reviewerID != user.Id {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		_, err = db.Exec("UPDATE reviews SET status = 'published', updated_at = NOW() WHERE id = $1", id)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func createPeriod(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		var input struct {
			Name      string `json:"name"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		start, err := time.Parse("2006-01-02", input.StartDate)
		if err != nil {
			http.Error(w, "Invalid start_date", http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02", input.EndDate)
		if err != nil {
			http.Error(w, "Invalid end_date", http.StatusBadRequest)
			return
		}
		if end.Before(start) {
			http.Error(w, "end_date must be >= start_date", http.StatusBadRequest)
			return
		}
		var id int64
		err = db.QueryRow("INSERT INTO review_periods (name, start_date, end_date) VALUES ($1, $2, $3) RETURNING id",
			input.Name, start, end).Scan(&id)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{"id": id})
	}
}

func updatePeriod(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "Invalid period ID", http.StatusBadRequest)
			return
		}
		var input struct {
			Name      string `json:"name"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		start, err := time.Parse("2006-01-02", input.StartDate)
		if err != nil {
			http.Error(w, "Invalid start_date", http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02", input.EndDate)
		if err != nil {
			http.Error(w, "Invalid end_date", http.StatusBadRequest)
			return
		}
		if end.Before(start) {
			http.Error(w, "end_date must be >= start_date", http.StatusBadRequest)
			return
		}
		_, err = db.Exec("UPDATE review_periods SET name=$1, start_date=$2, end_date=$3 WHERE id=$4",
			input.Name, start, end, id)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func deletePeriod(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := getCurrentUser(r, db)
		if err != nil || user.Role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		vars := mux.Vars(r)
		id, err := strconv.ParseInt(vars["id"], 10, 64)
		if err != nil {
			http.Error(w, "Invalid period ID", http.StatusBadRequest)
			return
		}
		_, err = db.Exec("DELETE FROM review_periods WHERE id=$1", id)
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func main() {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
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
		dbname = "tracker"
	}
	dsn := fmt.Sprintf("user=%s dbname=%s password=%s sslmode=disable host=%s port=%s",
		user, dbname, password, host, port)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	rtr := mux.NewRouter()
	rtr.HandleFunc("/", index).Methods("GET")
	rtr.HandleFunc("/users", usersPage(db)).Methods("GET")
	rtr.HandleFunc("/reviews", reviewsPage).Methods("GET")
	rtr.HandleFunc("/my-reviews", myReviewsPage).Methods("GET")
	rtr.HandleFunc("/reviews/{id:[0-9]+}", reviewDetailPage).Methods("GET")
	rtr.HandleFunc("/reviews/new", newReviewForm(db)).Methods("GET")
	rtr.HandleFunc("/reviews/{id:[0-9]+}/edit", editReviewForm(db)).Methods("GET")
	rtr.HandleFunc("/admin/periods", periodsListPage(db)).Methods("GET")
	rtr.HandleFunc("/admin/periods/new", newPeriodForm(db)).Methods("GET")
	rtr.HandleFunc("/admin/periods/{id:[0-9]+}/edit", editPeriodForm(db)).Methods("GET")

	api := rtr.PathPrefix("/api").Subrouter()
	api.HandleFunc("/me", apiMe(db)).Methods("GET")
	api.HandleFunc("/employees", apiEmployees(db)).Methods("GET")
	api.HandleFunc("/periods", apiPeriods(db)).Methods("GET")
	api.HandleFunc("/reviews", apiReviewsList(db)).Methods("GET")
	api.HandleFunc("/reviews/{id:[0-9]+}", apiReviewDetail(db)).Methods("GET")
	api.HandleFunc("/reviews", createReview(db)).Methods("POST")
	api.HandleFunc("/reviews/{id:[0-9]+}", updateReview(db)).Methods("PUT")
	api.HandleFunc("/reviews/{id:[0-9]+}/publish", publishReview(db)).Methods("POST")
	api.HandleFunc("/periods", createPeriod(db)).Methods("POST")
	api.HandleFunc("/periods/{id:[0-9]+}", updatePeriod(db)).Methods("PUT")
	api.HandleFunc("/periods/{id:[0-9]+}", deletePeriod(db)).Methods("DELETE")

	rtr.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", rtr))
}
