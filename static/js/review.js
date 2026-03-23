async function loadReview() {
    const path = window.location.pathname;
    const match = path.match(/\/reviews\/(\d+)/);
    if (!match) return;
    const id = match[1];

    try {
        // Загружаем детали отзыва
        const response = await apiFetch(`/api/reviews/${id}`);
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        const review = await response.json();

        // Загружаем информацию о текущем пользователе
        let canEdit = false, canPublish = false;
        try {
            const meResp = await apiFetch('/api/me');
            if (meResp.ok) {
                const me = await meResp.json();
                const isAdminManager = me.role === 'admin' || me.role === 'manager';
                if (isAdminManager && review.status === 'draft') {
                    canEdit = true;
                    canPublish = true;
                }
            }
        } catch (e) {
            console.warn('Cannot get current user', e);
        }

        // Рендерим HTML
        const container = document.getElementById('review-detail');
        if (!container) return;

        const html = `
            <dl class="row">
                <dt class="col-sm-3">ID</dt>
                <dd class="col-sm-9">${review.ID}</dd>
                <dt class="col-sm-3">Сотрудник</dt>
                <dd class="col-sm-9">${escapeHtml(review.EmployeeName)}</dd>
                <dt class="col-sm-3">Рецензент</dt>
                <dd class="col-sm-9">${escapeHtml(review.ReviewerName)}</dd>
                <dt class="col-sm-3">Период</dt>
                <dd class="col-sm-9">${escapeHtml(review.PeriodName)} (${review.PeriodStart} - ${review.PeriodEnd})</dd>
                <dt class="col-sm-3">Soft skills</dt>
                <dd class="col-sm-9">${review.SoftSkillsScore || '—'}</dd>
                <dt class="col-sm-3">Hard skills</dt>
                <dd class="col-sm-9">${review.HardSkillsScore || '—'}</dd>
                <dt class="col-sm-3">Комментарий</dt>
                <dd class="col-sm-9">${escapeHtml(review.Comment) || '—'}</dd>
                <dt class="col-sm-3">Статус</dt>
                <dd class="col-sm-9">${review.Status}</dd>
                <dt class="col-sm-3">Дата создания</dt>
                <dd class="col-sm-9">${formatDate(review.CreatedAt)}</dd>
                <dt class="col-sm-3">Дата обновления</dt>
                <dd class="col-sm-9">${formatDate(review.UpdatedAt)}</dd>
            </dl>
            <div class="mt-3">
                ${canEdit ? `<a href="/reviews/${review.ID}/edit" class="btn btn-warning me-2">Редактировать</a>` : ''}
                ${canPublish ? `<button class="btn btn-success" id="publishBtn">Опубликовать</button>` : ''}
                <a href="/reviews" class="btn btn-secondary">Назад к списку</a>
            </div>
        `;
        container.innerHTML = html;

        if (canPublish) {
            document.getElementById('publishBtn').addEventListener('click', async () => {
                if (!confirm('Опубликовать отзыв? После публикации редактирование будет недоступно.')) return;
                try {
                    const resp = await apiFetch(`/api/reviews/${review.ID}/publish`, { method: 'POST' });
                    if (resp.ok) {
                        window.location.reload();
                    } else {
                        alert('Ошибка при публикации');
                    }
                } catch(e) {
                    alert('Ошибка сети');
                }
            });
        }
    } catch (error) {
        console.error('Error loading review:', error);
        document.getElementById('review-detail').innerHTML = '<div class="alert alert-danger">Ошибка загрузки данных</div>';
    }
}

function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/[&<>]/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        return m;
    });
}

function formatDate(dateStr) {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return date.toLocaleString('ru-RU');
}

document.addEventListener('DOMContentLoaded', loadReview);