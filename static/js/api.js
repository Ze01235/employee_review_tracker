window.apiFetch = async function(url, options = {}) {
    const userId = localStorage.getItem('user_id');
    if (userId) {
        options.headers = {
            ...options.headers,
            'X-User-Id': userId
        };
    }
    const response = await fetch(url, options);
    return response;
};

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

function translateStatus(status) {
    if (status === 'draft') return 'Черновик';
    if (status === 'published') return 'Опубликовано';
    return status;
}