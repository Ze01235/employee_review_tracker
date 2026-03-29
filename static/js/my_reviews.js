document.addEventListener('DOMContentLoaded', async function() {
    const container = document.getElementById('my-reviews-container');
    if (!container) return;

    try {
        const response = await apiFetch('/api/reviews');
        if (!response.ok) throw new Error('HTTP ' + response.status);
        const reviews = await response.json();

        if (reviews.length === 0) {
            container.innerHTML = '<div class="alert alert-info">У вас пока нет опубликованных оценок.</div>';
            return;
        }

        let html = `
            <table class="table table-striped">
                <thead>
                    <tr>
                        <th>Период</th>
                        <th>Рецензент</th>
                        <th>Soft skills</th>
                        <th>Hard skills</th>
                        <th>Комментарий</th>
                        <th>Статус</th>
                        <th>Дата</th>
                        <th>Действия</th>
                    </tr>
                </thead>
                <tbody>
        `;
        for (const r of reviews) {
            html += `
                <tr>
                    <td>${escapeHtml(r.period_name)}</td>
                    <td>${escapeHtml(r.reviewer_name)}</td>
                    <td>${r.soft_skills_score || '—'}</td>
                    <td>${r.hard_skills_score || '—'}</td>
                    <td>${escapeHtml(r.comment) || '—'}</td>
                    <td>${translateStatus(r.status)}</td>
                    <td>${formatDate(r.created_at)}</td>
                    <td><a href="/reviews/${r.id}" class="btn btn-sm btn-primary">Просмотр</a></td>
                </tr>
            `;
        }
        html += `</tbody></table>`;
        container.innerHTML = html;
    } catch (err) {
        console.error(err);
        container.innerHTML = '<div class="alert alert-danger">Ошибка загрузки данных</div>';
    }
});