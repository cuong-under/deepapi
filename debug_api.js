const http = require('http');

// Step 1: Login
const loginData = JSON.stringify({ password: 'admin' });
const loginReq = http.request({
  hostname: 'localhost',
  port: 5001,
  path: '/admin/login',
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Content-Length': loginData.length
  }
}, (res) => {
  let data = '';
  res.on('data', chunk => data += chunk);
  res.on('end', () => {
    if (res.statusCode !== 200) {
      console.log('Login failed:', data);
      return;
    }
    
    const { token } = JSON.parse(data);
    console.log('Login success, token:', token.substring(0, 20) + '...');
    
    // Step 2: Call analytics API
    const analyticsReq = http.request({
      hostname: 'localhost',
      port: 5001,
      path: '/admin/analytics/overview',
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }, (res2) => {
      let data2 = '';
      res2.on('data', chunk => data2 += chunk);
      res2.on('end', () => {
        console.log('\nAnalytics API response:');
        console.log(JSON.stringify(JSON.parse(data2), null, 2));
      });
    });
    analyticsReq.end();
  });
});

loginReq.on('error', err => console.error('Error:', err));
loginReq.write(loginData);
loginReq.end();
