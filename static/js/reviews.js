let currentFilters = { period_id: '', employee_id: '' };

async function loadFilters() {
    try {
        const meResp = await apiFetch('/api/me');
        if (!meResp.ok) {
            if (meResp.status === 401) {
                window.location.href = '/users';
                return;
            }
            throw new Error('HTTP ' + meResp.status);
        }
        const me = await meResp.json();

        const periodsResp = await apiFetch('/api/periods');
        if (!periodsResp.ok) throw new Error('Periods fetch failed');
        const periods = await periodsResp.json();
        const periodSelect = document.getElementById('period_id');
        if (periodSelect) {
            periodSelect.innerHTML = '<option value="">Все периоды</option>' +
                periods.map(p => `<option value="${p.id}">${escapeHtml(p.name)} (${p.start_date} - ${p.end_date})</option>`).join('');
        }

        if (me.role === 'admin' || me.role === 'manager') {
            const empResp = await apiFetch('/api/employees');
            if (empResp.ok) {
                const employees = await empResp.json();
                const empSelect = document.getElementById('employee_id');
                if (empSelect) {
                    empSelect.innerHTML = '<option value="">Все сотрудники</option>' +
                        employees.map(e => `<option value="${e.id}">${escapeHtml(e.name)} (${e.role})</option>`).join('');
                    empSelect.disabled = false;
                }
            }
        } else {
            const empSelect = document.getElementById('employee_id');
            if (empSelect) empSelect.disabled = true;
        }

        const urlParams = new URLSearchParams(window.location.search);
        currentFilters.period_id = urlParams.get('period_id') || '';
        currentFilters.employee_id = urlParams.get('employee_id') || '';
        if (periodSelect) periodSelect.value = currentFilters.period_id;
        const empSelect = document.getElementById('employee_id');
        if (empSelect) empSelect.value = currentFilters.employee_id;

        loadReviews();
    } catch (err) {
        console.error('Error loading filters:', err);
        const container = document.getElementById('reviews-container');
        if (container) container.innerHTML = '<div class="alert alert-danger">Ошибка загрузки фильтров</div>';
    }
}

async function loadReviews() {
    const container = document.getElementById('reviews-container');
    if (!container) return;
    container.innerHTML = '<div class="text-center">Загрузка...</div>';

    try {
        let url = '/api/reviews';
        const params = new URLSearchParams();
        if (currentFilters.period_id) params.append('period_id', currentFilters.period_id);
        if (currentFilters.employee_id) params.append('employee_id', currentFilters.employee_id);
        if (params.toString()) url += '?' + params.toString();

        const response = await apiFetch(url);
        if (!response.ok) {
            if (response.status === 401) {
                window.location.href = '/users';
                return;
            }
            throw new Error('HTTP ' + response.status);
        }
        const reviews = await response.json();

        const meResp = await apiFetch('/api/me');
        let canCreate = false;
        if (meResp.ok) {
            const me = await meResp.json();
            canCreate = me.role === 'admin' || me.role === 'manager';
        }

        renderReviewsTable(reviews, canCreate);
    } catch (err) {
        console.error(err);
        container.innerHTML = '<div class="alert alert-danger">Ошибка загрузки отзывов</div>';
    }
}

function renderReviewsTable(reviews, canCreate) {
    const container = document.getElementById('reviews-container');
    if (!container) return;
    let html = `
        <div class="mb-3">
            <a href="/reviews/new" class="btn btn-primary" id="createReviewBtn" style="display: ${canCreate ? 'inline-block' : 'none'}">Создать отзыв</a>
        </div>
        <table class="table table-striped">
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Сотрудник</th>
                    <th>Рецензент</th>
                    <th>Период</th>
                    <th>Soft skills</th>
                    <th>Hard skills</th>
                    <th>Статус</th>
                    <th>Дата создания</th>
                    <th>Действия</th>
                </tr>
            </thead>
            <tbody>
    `;
    if (reviews.length === 0) {
        html += '<tr><td colspan="9" class="text-center">Нет данных</td></tr>';
    } else {
        reviews.forEach(r => {
            html += `
                <tr>
                    <td>${r.id}</td>
                    <td>${escapeHtml(r.employee_name)}</td>
                    <td>${escapeHtml(r.reviewer_name)}</td>
                    <td>${escapeHtml(r.period_name)}</td>
                    <td>${r.soft_skills_score || '—'}</td>
                    <td>${r.hard_skills_score || '—'}</td>
                    <td>${translateStatus(r.status)}</td>
                    <td>${formatDate(r.created_at)}</td>
                    <td><a href="/reviews/${r.id}" class="btn btn-sm btn-primary">Просмотр</a></td>
                </tr>
            `;
        });
    }
    html += `</tbody></table>`;
    container.innerHTML = html;
}

document.addEventListener('DOMContentLoaded', () => {
    const filterForm = document.getElementById('filter-form');
    if (filterForm) {
        filterForm.addEventListener('submit', (e) => {
            e.preventDefault();
            const periodId = document.getElementById('period_id')?.value || '';
            const employeeId = document.getElementById('employee_id')?.value || '';
            currentFilters = { period_id: periodId, employee_id: employeeId };

            const params = new URLSearchParams();
            if (periodId) params.append('period_id', periodId);
            if (employeeId) params.append('employee_id', employeeId);
            const newUrl = window.location.pathname + (params.toString() ? '?' + params.toString() : '');
            window.history.pushState({}, '', newUrl);
            loadReviews();
        });
    }
    loadFilters();
});