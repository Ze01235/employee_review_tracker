function selectUser(userId, userName, userRole) {
    localStorage.setItem('user_id', userId);
    localStorage.setItem('user_name', userName);
    localStorage.setItem('user_role', userRole);
    // Устанавливаем cookie с user_id на 1 день
    document.cookie = `user_id=${userId}; path=/; max-age=86400`;
    window.location.href = '/';
}

document.addEventListener('DOMContentLoaded', function () {
    const buttons = document.querySelectorAll('.select-user');
    buttons.forEach(btn => {
        btn.addEventListener('click', function (e) {
            e.stopPropagation(); // чтобы не сработал клик по карточке
            const card = this.closest('.user-card');
            const userId = card.dataset.userId;
            const userName = card.dataset.userName;
            const userRole = card.dataset.userRole;
            selectUser(userId, userName, userRole);
        });
    });

    const cards = document.querySelectorAll('.user-card');
    cards.forEach(card => {
        card.addEventListener('click', function () {
            const userId = this.dataset.userId;
            const userName = this.dataset.userName;
            const userRole = this.dataset.userRole;
            selectUser(userId, userName, userRole);
        });
    });
});

async function apiFetch(url, options = {}) {
    const userId = localStorage.getItem('user_id');
    if (userId) {
        options.headers = {
            ...options.headers,
            'X-User-Id': userId
        };
    }
    const response = await fetch(url, options);
    return response;
}

function selectUser(userId, userName, userRole) {
    localStorage.setItem('user_id', userId);
    localStorage.setItem('user_name', userName);
    localStorage.setItem('user_role', userRole);
    document.cookie = `user_id=${userId}; path=/; max-age=86400`;
    window.location.href = '/';
}

document.addEventListener('DOMContentLoaded', function () {
    const buttons = document.querySelectorAll('.select-user');
    buttons.forEach(btn => {
        btn.addEventListener('click', function (e) {
            e.stopPropagation();
            const card = this.closest('.user-card');
            selectUser(card.dataset.userId, card.dataset.userName, card.dataset.userRole);
        });
    });
    const cards = document.querySelectorAll('.user-card');
    cards.forEach(card => {
        card.addEventListener('click', function () {
            selectUser(this.dataset.userId, this.dataset.userName, this.dataset.userRole);
        });
    });
});

