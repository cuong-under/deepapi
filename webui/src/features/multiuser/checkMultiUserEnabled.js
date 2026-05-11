// Check if multi-user mode is enabled by checking if the API endpoint exists
export async function checkMultiUserEnabled() {
    try {
        const res = await fetch('/api/auth/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username: '__check__', password: '__check__' })
        })

        // If endpoint exists (even if returns error), multi-user is enabled
        // 404 means endpoint doesn't exist = multi-user disabled
        return res.status !== 404
    } catch (e) {
        return false
    }
}
