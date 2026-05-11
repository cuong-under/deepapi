#!/bin/bash

# Test script for Multi-User Authentication System

BASE_URL="http://localhost:5001"
API_URL="$BASE_URL/api"

echo "=== DS2API Multi-User Authentication Test ==="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Register new user
echo -e "${YELLOW}Test 1: Register new user${NC}"
REGISTER_RESPONSE=$(curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }')

echo "$REGISTER_RESPONSE" | jq .

if echo "$REGISTER_RESPONSE" | jq -e '.token' > /dev/null; then
  echo -e "${GREEN}✓ Registration successful${NC}"
  USER_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.token')
else
  echo -e "${RED}✗ Registration failed${NC}"
  exit 1
fi
echo ""

# Test 2: Login with admin
echo -e "${YELLOW}Test 2: Login with admin${NC}"
ADMIN_LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }')

echo "$ADMIN_LOGIN_RESPONSE" | jq .

if echo "$ADMIN_LOGIN_RESPONSE" | jq -e '.token' > /dev/null; then
  echo -e "${GREEN}✓ Admin login successful${NC}"
  ADMIN_TOKEN=$(echo "$ADMIN_LOGIN_RESPONSE" | jq -r '.token')
else
  echo -e "${RED}✗ Admin login failed${NC}"
  exit 1
fi
echo ""

# Test 3: Get current user info
echo -e "${YELLOW}Test 3: Get current user info${NC}"
ME_RESPONSE=$(curl -s -X GET "$API_URL/auth/me" \
  -H "Authorization: Bearer $USER_TOKEN")

echo "$ME_RESPONSE" | jq .

if echo "$ME_RESPONSE" | jq -e '.username' > /dev/null; then
  echo -e "${GREEN}✓ Get user info successful${NC}"
else
  echo -e "${RED}✗ Get user info failed${NC}"
fi
echo ""

# Test 4: Create account for user
echo -e "${YELLOW}Test 4: Create account for user${NC}"
CREATE_ACCOUNT_RESPONSE=$(curl -s -X POST "$API_URL/user/accounts" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Account",
    "remark": "My test account",
    "email": "myaccount@deepseek.com",
    "password": "deepseek_password"
  }')

echo "$CREATE_ACCOUNT_RESPONSE" | jq .

if echo "$CREATE_ACCOUNT_RESPONSE" | jq -e '.id' > /dev/null; then
  echo -e "${GREEN}✓ Create account successful${NC}"
  ACCOUNT_ID=$(echo "$CREATE_ACCOUNT_RESPONSE" | jq -r '.id')
else
  echo -e "${RED}✗ Create account failed${NC}"
fi
echo ""

# Test 5: List user accounts
echo -e "${YELLOW}Test 5: List user accounts${NC}"
LIST_ACCOUNTS_RESPONSE=$(curl -s -X GET "$API_URL/user/accounts" \
  -H "Authorization: Bearer $USER_TOKEN")

echo "$LIST_ACCOUNTS_RESPONSE" | jq .

if echo "$LIST_ACCOUNTS_RESPONSE" | jq -e '.accounts' > /dev/null; then
  echo -e "${GREEN}✓ List accounts successful${NC}"
else
  echo -e "${RED}✗ List accounts failed${NC}"
fi
echo ""

# Test 6: Create API key
echo -e "${YELLOW}Test 6: Create API key${NC}"
CREATE_KEY_RESPONSE=$(curl -s -X POST "$API_URL/user/keys" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test API Key",
    "remark": "For testing"
  }')

echo "$CREATE_KEY_RESPONSE" | jq .

if echo "$CREATE_KEY_RESPONSE" | jq -e '.api_key' > /dev/null; then
  echo -e "${GREEN}✓ Create API key successful${NC}"
  API_KEY_ID=$(echo "$CREATE_KEY_RESPONSE" | jq -r '.id')
else
  echo -e "${RED}✗ Create API key failed${NC}"
fi
echo ""

# Test 7: List API keys
echo -e "${YELLOW}Test 7: List API keys${NC}"
LIST_KEYS_RESPONSE=$(curl -s -X GET "$API_URL/user/keys" \
  -H "Authorization: Bearer $USER_TOKEN")

echo "$LIST_KEYS_RESPONSE" | jq .

if echo "$LIST_KEYS_RESPONSE" | jq -e '.keys' > /dev/null; then
  echo -e "${GREEN}✓ List API keys successful${NC}"
else
  echo -e "${RED}✗ List API keys failed${NC}"
fi
echo ""

# Test 8: Admin can see all accounts
echo -e "${YELLOW}Test 8: Admin can see all accounts${NC}"
ADMIN_LIST_ACCOUNTS=$(curl -s -X GET "$API_URL/user/accounts" \
  -H "Authorization: Bearer $ADMIN_TOKEN")

echo "$ADMIN_LIST_ACCOUNTS" | jq .

if echo "$ADMIN_LIST_ACCOUNTS" | jq -e '.accounts' > /dev/null; then
  echo -e "${GREEN}✓ Admin list accounts successful${NC}"
else
  echo -e "${RED}✗ Admin list accounts failed${NC}"
fi
echo ""

# Test 9: Update account
if [ ! -z "$ACCOUNT_ID" ]; then
  echo -e "${YELLOW}Test 9: Update account${NC}"
  UPDATE_ACCOUNT_RESPONSE=$(curl -s -X PUT "$API_URL/user/accounts/$ACCOUNT_ID" \
    -H "Authorization: Bearer $USER_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "Updated Account",
      "remark": "Updated remark",
      "email": "updated@deepseek.com",
      "password": "new_password"
    }')

  echo "$UPDATE_ACCOUNT_RESPONSE" | jq .

  if echo "$UPDATE_ACCOUNT_RESPONSE" | jq -e '.message' > /dev/null; then
    echo -e "${GREEN}✓ Update account successful${NC}"
  else
    echo -e "${RED}✗ Update account failed${NC}"
  fi
  echo ""
fi

# Test 10: Logout
echo -e "${YELLOW}Test 10: Logout${NC}"
LOGOUT_RESPONSE=$(curl -s -X POST "$API_URL/auth/logout" \
  -H "Authorization: Bearer $USER_TOKEN")

echo "$LOGOUT_RESPONSE" | jq .

if echo "$LOGOUT_RESPONSE" | jq -e '.message' > /dev/null; then
  echo -e "${GREEN}✓ Logout successful${NC}"
else
  echo -e "${RED}✗ Logout failed${NC}"
fi
echo ""

# Test 11: Try to access with logged out token (should fail)
echo -e "${YELLOW}Test 11: Access with logged out token (should fail)${NC}"
AFTER_LOGOUT_RESPONSE=$(curl -s -X GET "$API_URL/auth/me" \
  -H "Authorization: Bearer $USER_TOKEN")

echo "$AFTER_LOGOUT_RESPONSE"

if echo "$AFTER_LOGOUT_RESPONSE" | grep -q "error"; then
  echo -e "${GREEN}✓ Token correctly invalidated after logout${NC}"
else
  echo -e "${RED}✗ Token still valid after logout${NC}"
fi
echo ""

echo -e "${GREEN}=== All tests completed ===${NC}"
