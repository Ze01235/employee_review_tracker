document.addEventListener('DOMContentLoaded', async function() {
    const greetingEl = document.getElementById('greeting');
    if (!greetingEl) return;

    try {
        const response = await apiFetch('/api/me');
        if (response.ok) {
            const user = await response.json();
            greetingEl.textContent = `Добро пожаловать, ${user.name}!`;
        } else if (response.status === 401) {
            greetingEl.textContent = 'Добро пожаловать, Гость!';
        } else {
            greetingEl.textContent = 'Ошибка загрузки данных';
        }
    } catch (error) {
        console.error('Fetch error:', error);
        greetingEl.textContent = 'Ошибка соединения';
    }
});