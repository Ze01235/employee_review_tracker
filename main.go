package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
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
	ID        int64
	Name      string
	StartDate string
	EndDate   string
}

type Review struct {
	ID              int64
	EmployeeID      int64
	EmployeeName    string
	ReviewerID      int64
	ReviewerName    string
	PeriodID        int64
	PeriodName      string
	PeriodStart     string
	PeriodEnd       string
	SoftSkillsScore int
	HardSkillsScore int
	Comment         string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ReviewInput struct {
	EmployeeID      int64  `json:"employee_id"`
	ReviewerID      int64  `json:"reviewer_id"`
	PeriodID        int64  `json:"period_id"`
	SoftSkillsScore int    `json:"soft_skills_score"`
	HardSkillsScore int    `json:"hard_skills_score"`
	Comment         string `json:"comment"`
}

type ReviewDetailData struct {
	Review
	CanEdit    bool
	CanPublish bool
}

type ReviewsListData struct {
	Reviews            []Review
	Periods            []Period
	Employees          []User
	SelectedPeriodID   string
	SelectedEmployeeID string
	CanCreate          bool
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
	return periods, nil
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
	return users, nil
}

func index(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "index", nil) // данные не передаём, подгрузим через API
}

// users отдаёт страницу выбора пользователя
func users(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("templates/users.html", "templates/header.html", "templates/footer.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rows, err := db.Query("SELECT id, name, role FROM users")
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.Id, &u.Name, &u.Role); err != nil {
				http.Error(w, "Scan error", http.StatusInternalServerError)
				return
			}
			users = append(users, u)
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

func reviewDetailPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/review.html", "templates/header.html", "templates/footer.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "review", nil)
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
	// Проверка существования employee_id, reviewer_id, period_id может быть выполнена запросом к БД
	return nil
}

func createReview(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка прав
		user, err := getCurrentUser(r, db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !checkAdminManager(user) {
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

		// Дополнительно проверим существование связанных записей
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", input.EmployeeID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "Employee not found", http.StatusBadRequest)
			return
		}
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", input.ReviewerID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "Reviewer not found", http.StatusBadRequest)
			return
		}
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM review_periods WHERE id = $1)", input.PeriodID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "Period not found", http.StatusBadRequest)
			return
		}

		// Вставка
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
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !checkAdminManager(user) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid review ID", http.StatusBadRequest)
			return
		}

		// Проверим, что review существует и имеет статус draft
		var status string
		err = db.QueryRow("SELECT status FROM reviews WHERE id = $1", id).Scan(&status)
		if err == sql.ErrNoRows {
			http.Error(w, "Review not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if status != "draft" {
			http.Error(w, "Cannot edit published review", http.StatusForbidden)
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

		// Обновляем поля (не меняем employee_id, reviewer_id, period_id – можно разрешить, но для простоты оставим)
		// Решим, что можно менять только оценки и комментарий. Если нужно менять и другие, добавим их в SET.
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
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !checkAdminManager(user) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		vars := mux.Vars(r)
		idStr := vars["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid review ID", http.StatusBadRequest)
			return
		}

		var status string
		err = db.QueryRow("SELECT status FROM reviews WHERE id = $1", id).Scan(&status)
		if err == sql.ErrNoRows {
			http.Error(w, "Review not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if status != "draft" {
			http.Error(w, "Review already published", http.StatusForbidden)
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

func newReviewForm(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// проверка прав не нужна, так как права проверяются при отправке формы через API
		employees, err := getEmployees(db)
		if err != nil {
			http.Error(w, "Error loading employees", http.StatusInternalServerError)
			return
		}
		periods, err := getPeriods(db)
		if err != nil {
			http.Error(w, "Error loading periods", http.StatusInternalServerError)
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
		idStr := vars["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid review ID", http.StatusBadRequest)
			return
		}
		// Получаем данные review
		var review Review
		var softScore, hardScore sql.NullInt64
		var comment sql.NullString
		err = db.QueryRow(`
            SELECT id, employee_id, reviewer_id, period_id, soft_skills_score, hard_skills_score, comment, status
            FROM reviews WHERE id = $1
        `, id).Scan(
			&review.ID, &review.EmployeeID, &review.ReviewerID, &review.PeriodID,
			&softScore, &hardScore, &comment, &review.Status,
		)
		if err == sql.ErrNoRows {
			http.Error(w, "Review not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if review.Status != "draft" {
			http.Error(w, "Cannot edit published review", http.StatusForbidden)
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
			http.Error(w, "Error loading employees", http.StatusInternalServerError)
			return
		}
		periods, err := getPeriods(db)
		if err != nil {
			http.Error(w, "Error loading periods", http.StatusInternalServerError)
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

// apiReviewsList возвращает JSON со списком отзывов с учётом фильтров
func apiReviewsList(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		periodID := r.URL.Query().Get("period_id")
		employeeID := r.URL.Query().Get("employee_id")

		// Формируем запрос (аналогично reviewsList)
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

		if periodID != "" {
			query += fmt.Sprintf(" AND r.period_id = $%d", argCounter)
			args = append(args, periodID)
			argCounter++
		}
		if employeeID != "" {
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
			err := rows.Scan(
				&r.ID, &r.EmployeeID, &r.EmployeeName,
				&r.ReviewerID, &r.ReviewerName,
				&r.PeriodID, &r.PeriodName,
				&softScore, &hardScore,
				&comment, &r.Status, &r.CreatedAt, &r.UpdatedAt,
			)
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reviews)
	}
}

// apiReviewDetail возвращает JSON с деталями отзыва
func apiReviewDetail(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			http.Error(w, "Missing review ID", http.StatusBadRequest)
			return
		}

		query := `
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
        `
		var review Review
		var softScore, hardScore sql.NullInt64
		var comment sql.NullString
		var startDate, endDate time.Time
		err := db.QueryRow(query, id).Scan(
			&review.ID, &review.EmployeeID, &review.EmployeeName,
			&review.ReviewerID, &review.ReviewerName,
			&review.PeriodID, &review.PeriodName,
			&startDate, &endDate,
			&softScore, &hardScore,
			&comment, &review.Status, &review.CreatedAt, &review.UpdatedAt,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Review not found", http.StatusNotFound)
			} else {
				http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
			}
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

// apiMe возвращает данные текущего пользователя по заголовку X-User-Id
func apiMe(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-Id")
		if userID == "" {
			http.Error(w, "Missing X-User-Id header", http.StatusUnauthorized)
			return
		}

		var user User
		err := db.QueryRow("SELECT id, name, role FROM users WHERE id = $1", userID).
			Scan(&user.Id, &user.Name, &user.Role)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "User not found", http.StatusNotFound)
			} else {
				http.Error(w, "Database error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func main() {
	dsn := "user=postgres dbname=tracker password=root sslmode=disable"
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
	rtr.HandleFunc("/users", users(db)).Methods("GET")
	rtr.HandleFunc("/api/me", apiMe(db)).Methods("GET")

	rtr.HandleFunc("/reviews", reviewsPage).Methods("GET")
	rtr.HandleFunc("/reviews/{id:[0-9]+}", reviewDetailPage).Methods("GET")
	rtr.HandleFunc("/reviews/new", newReviewForm(db)).Methods("GET")
	rtr.HandleFunc("/reviews/{id:[0-9]+}/edit", editReviewForm(db)).Methods("GET")

	// API для данных
	api := rtr.PathPrefix("/api").Subrouter()
	api.HandleFunc("/reviews", apiReviewsList(db)).Methods("GET")
	api.HandleFunc("/reviews/{id:[0-9]+}", apiReviewDetail(db)).Methods("GET")
	api.HandleFunc("/reviews", createReview(db)).Methods("POST")
	api.HandleFunc("/reviews/{id:[0-9]+}", updateReview(db)).Methods("PUT")
	api.HandleFunc("/reviews/{id:[0-9]+}/publish", publishReview(db)).Methods("POST")
	api.HandleFunc("/employees", apiEmployees(db)).Methods("GET")
	api.HandleFunc("/periods", apiPeriods(db)).Methods("GET")

	rtr.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", rtr))
}
