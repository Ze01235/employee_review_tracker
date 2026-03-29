(async function() {
    const titleEl = document.getElementById('form-title');
    const container = document.getElementById('review-form-container');
    if (!container) return; // не страница формы

    const path = window.location.pathname;
    const match = path.match(/\/reviews\/(\d+)\/edit/);
    const isEdit = match !== null;
    const reviewId = isEdit ? match[1] : null;

    try {
        const [employees, periods] = await Promise.all([
            apiFetch('/api/employees').then(r => r.json()),
            apiFetch('/api/periods').then(r => r.json())
        ]);

        let reviewData = null;
        if (isEdit) {
            const reviewResp = await apiFetch(`/api/reviews/${reviewId}`);
            if (!reviewResp.ok) throw new Error(`Не удалось загрузить отзыв: ${reviewResp.status}`);
            reviewData = await reviewResp.json();
            if (titleEl) titleEl.textContent = 'Редактирование отзыва';
        } else {
            if (titleEl) titleEl.textContent = 'Создание отзыва';
        }

        let optionsEmployees = employees.map(e => `<option value="${e.id}" ${reviewData && reviewData.EmployeeID === e.id ? 'selected' : ''}>${escapeHtml(e.name)} (${e.role})</option>`).join('');
        let optionsReviewers = employees.map(e => `<option value="${e.id}" ${reviewData && reviewData.ReviewerID === e.id ? 'selected' : ''}>${escapeHtml(e.name)} (${e.role})</option>`).join('');
        let optionsPeriods = periods.map(p => `<option value="${p.id}" ${reviewData && reviewData.PeriodID === p.id ? 'selected' : ''}>${escapeHtml(p.name)} (${p.start_date} - ${p.end_date})</option>`).join('');

        container.innerHTML = `
            <form id="reviewForm">
                <div class="mb-3">
                    <label for="employee_id" class="form-label">Сотрудник</label>
                    <select class="form-select" id="employee_id" name="employee_id" required>
                        <option value="">Выберите сотрудника</option>
                        ${optionsEmployees}
                    </select>
                </div>
                <div class="mb-3">
                    <label for="reviewer_id" class="form-label">Рецензент</label>
                    <select class="form-select" id="reviewer_id" name="reviewer_id" required>
                        <option value="">Выберите рецензента</option>
                        ${optionsReviewers}
                    </select>
                </div>
                <div class="mb-3">
                    <label for="period_id" class="form-label">Период</label>
                    <select class="form-select" id="period_id" name="period_id" required>
                        <option value="">Выберите период</option>
                        ${optionsPeriods}
                    </select>
                </div>
                <div class="mb-3">
                    <label for="soft_skills_score" class="form-label">Soft skills (1-5)</label>
                    <input type="number" class="form-control" id="soft_skills_score" name="soft_skills_score" min="1" max="5" value="${reviewData ? reviewData.soft_skills_score : ''}" required>
                </div>
                <div class="mb-3">
                    <label for="hard_skills_score" class="form-label">Hard skills (1-5)</label>
                    <input type="number" class="form-control" id="hard_skills_score" name="hard_skills_score" min="1" max="5" value="${reviewData ? reviewData.hard_skills_score : ''}" required>
                </div>
                <div class="mb-3">
                    <label for="comment" class="form-label">Комментарий</label>
                    <textarea class="form-control" id="comment" name="comment" rows="3" required>${reviewData ? escapeHtml(reviewData.comment) : ''}</textarea>
                </div>
                <button type="submit" class="btn btn-primary">Сохранить</button>
                <a href="/reviews" class="btn btn-outline-primary">Отмена</a>
            </form>
        `;

        document.getElementById('reviewForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            const formData = new FormData(this);
            const data = Object.fromEntries(formData.entries());
            data.employee_id = parseInt(data.employee_id);
            data.reviewer_id = parseInt(data.reviewer_id);
            data.period_id = parseInt(data.period_id);
            data.soft_skills_score = parseInt(data.soft_skills_score);
            data.hard_skills_score = parseInt(data.hard_skills_score);

            let url, method;
            if (isEdit) {
                url = `/api/reviews/${reviewId}`;
                method = 'PUT';
            } else {
                url = '/api/reviews';
                method = 'POST';
            }

            const resp = await apiFetch(url, {
                method: method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });

            if (resp.ok) {
                if (isEdit) {
                    window.location.href = `/reviews/${reviewId}`;
                } else {
                    const result = await resp.json();
                    window.location.href = `/reviews/${result.id}`;
                }
            } else {
                const errorText = await resp.text();
                alert('Ошибка: ' + errorText);
            }
        });
    } catch (error) {
        console.error(error);
        container.innerHTML = `<div class="alert alert-danger">Ошибка загрузки данных: ${error.message}</div>`;
    }
})();