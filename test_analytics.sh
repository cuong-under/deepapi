#!/bin/bash
# Login to get token
TOKEN=$(curl -s -X POST http://localhost:5001/admin/login \
  -H "Content-Type: application/json" \
  -d '{"password":"admin"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

echo "Token: ${TOKEN:0:20}..."

# Test analytics API
echo -e "\n=== Testing /admin/analytics/overview ==="
curl -s -X GET http://localhost:5001/admin/analytics/overview \
  -H "Authorization: Bearer $TOKEN" | head -100
