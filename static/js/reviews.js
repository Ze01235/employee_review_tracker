// Функция для форматирования даты
function formatDate(dateStr) {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return date.toLocaleString('ru-RU');
}

// Рендеринг таблицы
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
        html += `<tr><td colspan="9" class="text-center">Нет данных</td></tr>`;
    } else {
        reviews.forEach(r => {
            html += `
                <tr>
                    <td>${r.ID}</td>
                    <td>${escapeHtml(r.EmployeeName)}</td>
                    <td>${escapeHtml(r.ReviewerName)}</td>
                    <td>${escapeHtml(r.PeriodName)}</td>
                    <td>${r.SoftSkillsScore || '—'}</td>
                    <td>${r.HardSkillsScore || '—'}</td>
                    <td>${r.Status}</td>
                    <td>${formatDate(r.CreatedAt)}</td>
                    <td><a href="/reviews/${r.ID}" class="btn btn-sm btn-info">Просмотр</a></td>
                </tr>
            `;
        });
    }

    html += `</tbody></table>`;
    container.innerHTML = html;
}

// Простая защита от XSS
function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/[&<>]/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        return m;
    });
}

// Загрузка данных
async function loadReviews() {
    try {
        // Сначала проверим права текущего пользователя, чтобы показать кнопку создания
        let canCreate = false;
        try {
            const meResp = await apiFetch('/api/me');
            if (meResp.ok) {
                const me = await meResp.json();
                canCreate = me.role === 'admin' || me.role === 'manager';
            }
        } catch(e) {
            console.warn('Cannot get current user', e);
        }

        // Загружаем список отзывов
        const response = await apiFetch('/api/reviews');
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        const reviews = await response.json();
        renderReviewsTable(reviews, canCreate);
    } catch (error) {
        console.error('Error loading reviews:', error);
        document.getElementById('reviews-container').innerHTML = '<div class="alert alert-danger">Ошибка загрузки данных</div>';
    }
}

document.addEventListener('DOMContentLoaded', loadReviews);