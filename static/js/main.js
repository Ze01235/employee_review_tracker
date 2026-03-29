document.addEventListener('DOMContentLoaded', async function() {
    if (window.location.pathname === '/users') {
        return;
    }

    const greetingEl = document.getElementById('greeting');
    const myReviewsLink = document.getElementById('myReviewsLink');
    const adminPeriodsLink = document.getElementById('adminPeriodsLink');
    const userInfoSpan = document.getElementById('userInfo');

    function setNavLinks(role) {
        if (!myReviewsLink || !adminPeriodsLink) return;
        myReviewsLink.style.display = 'none';
        adminPeriodsLink.style.display = 'none';
        if (role === 'employee') {
            myReviewsLink.style.display = 'block';
        } else if (role === 'admin') {
            adminPeriodsLink.style.display = 'block';
            myReviewsLink.style.display = 'block';
        } else if (role === 'manager') {
            myReviewsLink.style.display = 'block';
        }
    }

    try {
        const response = await apiFetch('/api/me');
        if (response.ok) {
            const user = await response.json();
            if (greetingEl) greetingEl.textContent = `Добро пожаловать, ${user.name}!`;
            setNavLinks(user.role);
            if (userInfoSpan) userInfoSpan.textContent = `${user.name} (${user.role})`;
        } else if (response.status === 401) {
            window.location.href = '/users';
        } else {
            if (greetingEl) greetingEl.textContent = 'Ошибка загрузки данных';
        }
    } catch (error) {
        console.error('Fetch error in main.js:', error);
        if (greetingEl) greetingEl.textContent = 'Ошибка соединения';
    }
});