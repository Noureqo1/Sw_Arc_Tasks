#!/usr/bin/env bash

BASE_URL="http://localhost:8082"

info() {
  echo -e "\n=== $1 ==="
}

RANDOM_PART=$((RANDOM % 10000))
USER_EMAIL="user${RANDOM_PART}@example.com"
USER_PASS="secretpass"

info "Register user: $USER_EMAIL"
REGISTER_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASS}\"}" \
  "${BASE_URL}/auth/register")

echo "REGISTER_RESPONSE: $REGISTER_RESPONSE"

info "Login user to get JWT"
LOGIN_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -d "{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASS}\"}" \
  "${BASE_URL}/auth/login")

echo "LOGIN_RESPONSE: $LOGIN_RESPONSE"

ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.access_token')
if [[ -z "$ACCESS_TOKEN" || "$ACCESS_TOKEN" == "null" ]]; then
  echo "ERROR: Failed to obtain access_token!"
  exit 1
fi
echo "Got access_token: $ACCESS_TOKEN"

info "Create function with Python snippet"
CREATE_FUNC_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -d '{"owner":"alice","code":"print(\"SWAPD\")","language":"python"}' \
  "${BASE_URL}/functions")

echo "CREATE_FUNC_RESPONSE: $CREATE_FUNC_RESPONSE"

FUNCTION_ID=$(echo "$CREATE_FUNC_RESPONSE" | jq -r '.function_id')
if [[ -z "$FUNCTION_ID" || "$FUNCTION_ID" == "null" ]]; then
  echo "ERROR: Could not extract function_id!"
  exit 1
fi
echo "Function ID: $FUNCTION_ID"

info "Execute function: $FUNCTION_ID"
EXEC_RESPONSE=$(curl -s -X POST \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  "${BASE_URL}/functions/${FUNCTION_ID}/execute")

echo "EXEC_RESPONSE: $EXEC_RESPONSE"

JOB_ID=$(echo "$EXEC_RESPONSE" | jq -r '.job_id')
if [[ -z "$JOB_ID" || "$JOB_ID" == "null" ]]; then
  echo "ERROR: Could not extract job_id!"
  exit 1
fi
echo "Job ID: $JOB_ID"

info "Polling job until status=done or error"
while true; do
  JOB_INFO=$(curl -s -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    "${BASE_URL}/jobs/${JOB_ID}")

  STATUS=$(echo "$JOB_INFO" | jq -r '.Status')
  echo "Current job status: $STATUS"
  echo "$JOB_INFO"

  if [[ "$STATUS" == "done" || "$STATUS" == "error" ]]; then
    echo "Final job info:"
    echo "$JOB_INFO" | jq .
    break
  fi

  sleep 2
done


