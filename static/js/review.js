async function loadReview() {
    const container = document.getElementById('review-detail');
    if (!container) return;

    const path = window.location.pathname;
    const match = path.match(/\/reviews\/(\d+)/);
    if (!match) return;
    const id = match[1];

    try {
        const response = await apiFetch(`/api/reviews/${id}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const review = await response.json();

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

        const html = `
            <dl class="row">
                <dt class="col-sm-3">ID</dt>
                <dd class="col-sm-9">${review.id}</dd>
                <dt class="col-sm-3">Сотрудник</dt>
                <dd class="col-sm-9">${escapeHtml(review.employee_name)}</dd>
                <dt class="col-sm-3">Рецензент</dt>
                <dd class="col-sm-9">${escapeHtml(review.reviewer_name)}</dd>
                <dt class="col-sm-3">Период</dt>
                <dd class="col-sm-9">${escapeHtml(review.period_name)} (${review.period_start} - ${review.period_end})</dd>
                <dt class="col-sm-3">Soft skills</dt>
                <dd class="col-sm-9">${review.soft_skills_score || '—'}</dd>
                <dt class="col-sm-3">Hard skills</dt>
                <dd class="col-sm-9">${review.hard_skills_score || '—'}</dd>
                <dt class="col-sm-3">Комментарий</dt>
                <dd class="col-sm-9">${escapeHtml(review.comment) || '—'}</dd>
                <dt class="col-sm-3">Статус</dt>
                <dd class="col-sm-9">${translateStatus(review.status)}</dd>
                <dt class="col-sm-3">Дата создания</dt>
                <dd class="col-sm-9">${formatDate(review.created_at)}</dd>
                <dt class="col-sm-3">Дата обновления</dt>
                <dd class="col-sm-9">${formatDate(review.updated_at)}</dd>
            </dl>
            <div class="mt-3">
                ${canEdit ? `<a href="/reviews/${review.id}/edit" class="btn btn-secondary">Редактировать</a>` : ''}
                ${canPublish ? `<button class="btn btn-primary" id="publishBtn">Опубликовать</button>` : ''}
                <a href="/reviews" class="btn btn-outline-primary">Назад к списку</a>
            </div>
        `;
        container.innerHTML = html;

        if (canPublish) {
            document.getElementById('publishBtn').addEventListener('click', async () => {
                if (!confirm('Опубликовать отзыв? После публикации редактирование будет недоступно.')) return;
                try {
                    const resp = await apiFetch(`/api/reviews/${review.id}/publish`, { method: 'POST' });
                    if (resp.ok) {
                        window.location.reload();
                    } else {
                        alert('Ошибка при публикации');
                    }
                } catch (e) {
                    alert('Ошибка сети');
                }
            });
        }
    } catch (error) {
        console.error('Error loading review:', error);
        container.innerHTML = '<div class="alert alert-danger">Ошибка загрузки данных</div>';
    }
}

document.addEventListener('DOMContentLoaded', loadReview);