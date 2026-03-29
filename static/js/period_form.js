document.addEventListener('DOMContentLoaded', async function() {
    const form = document.getElementById('periodForm');
    if (!form) return;

    const path = window.location.pathname;
    const match = path.match(/\/admin\/periods\/(\d+)\/edit/);
    const isEdit = match !== null;
    const periodId = isEdit ? match[1] : null;

    form.addEventListener('submit', async function(e) {
        e.preventDefault();
        const name = document.getElementById('name').value.trim();
        const start_date = document.getElementById('start_date').value;
        const end_date = document.getElementById('end_date').value;

        if (!name || !start_date || !end_date) {
            alert('Заполните все поля');
            return;
        }
        if (new Date(end_date) < new Date(start_date)) {
            alert('Дата окончания не может быть раньше даты начала');
            return;
        }

        const data = { name, start_date, end_date };
        let url, method;
        if (isEdit) {
            url = `/api/periods/${periodId}`;
            method = 'PUT';
        } else {
            url = '/api/periods';
            method = 'POST';
        }

        try {
            const resp = await apiFetch(url, {
                method: method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            if (resp.ok) {
                window.location.href = '/admin/periods';
            } else {
                const text = await resp.text();
                alert('Ошибка: ' + text);
            }
        } catch (err) {
            alert('Ошибка сети');
        }
    });
});