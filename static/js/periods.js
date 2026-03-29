document.addEventListener('DOMContentLoaded', async function() {
    const container = document.getElementById('periods-container');
    if (!container) return;

    try {
        const response = await apiFetch('/api/periods');
        if (!response.ok) throw new Error('HTTP ' + response.status);
        const periods = await response.json();

        if (periods.length === 0) {
            container.innerHTML = '<div class="alert alert-info">Нет периодов оценки.</div>';
            return;
        }

        let html = `
            <table class="table table-striped">
                <thead>
                    <tr>
                        <th>Название</th>
                        <th>Дата начала</th>
                        <th>Дата окончания</th>
                        <th>Действия</th>
                    </tr>
                </thead>
                <tbody>
        `;
        for (const p of periods) {
            html += `
                <tr>
                    <td>${escapeHtml(p.name)}</td>
                    <td>${escapeHtml(p.start_date)}</td>
                    <td>${escapeHtml(p.end_date)}</td>
                    <td>
                        <a href="/admin/periods/${p.id}/edit" class="btn btn-sm btn-primary me-2">Редактировать</a>
                        <button class="btn btn-sm btn-outline-primary delete-period" data-id="${p.id}">Удалить</button>
                    </td>
                </tr>
            `;
        }
        html += `</tbody></table>`;
        container.innerHTML = html;

        // Обработчики удаления
        document.querySelectorAll('.delete-period').forEach(btn => {
            btn.addEventListener('click', async function() {
                const id = this.dataset.id;
                if (!confirm('Удалить период? Все связанные отзывы также будут удалены.')) return;
                try {
                    const resp = await apiFetch(`/api/periods/${id}`, { method: 'DELETE' });
                    if (resp.ok) {
                        window.location.reload();
                    } else {
                        alert('Ошибка при удалении');
                    }
                } catch (err) {
                    alert('Ошибка сети');
                }
            });
        });
    } catch (err) {
        console.error(err);
        container.innerHTML = '<div class="alert alert-danger">Ошибка загрузки данных</div>';
    }
});