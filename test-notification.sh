#!/bin/bash

# Test script for Pushem notification service

set -e

BASE_URL="${PUSHEM_URL:-http://localhost:8080}"
TOPIC="${1:-test-topic}"

echo "Testing Pushem Notification Service"
echo "===================================="
echo ""

echo "1. Checking if server is running..."
if curl -s -f "$BASE_URL/vapid-public-key" > /dev/null; then
    echo "   ✓ Server is responding"
else
    echo "   ✗ Server is not responding at $BASE_URL"
    exit 1
fi

echo ""
echo "2. Getting VAPID public key..."
PUBLIC_KEY=$(curl -s "$BASE_URL/vapid-public-key" | grep -o '"publicKey":"[^"]*"' | cut -d'"' -f4)
echo "   ✓ Public key: ${PUBLIC_KEY:0:20}..."

echo ""
echo "3. Sending test notification to topic '$TOPIC'..."
RESPONSE=$(curl -s -X POST "$BASE_URL/publish/$TOPIC" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Test Notification",
        "message": "This is a test from the Pushem test script!",
        "click_url": "https://github.com"
    }')

echo "   Response: $RESPONSE"

SENT=$(echo "$RESPONSE" | grep -o '"sent":[0-9]*' | cut -d':' -f2)
if [ "$SENT" = "0" ]; then
    echo "   ℹ No subscribers found for topic '$TOPIC'"
    echo ""
    echo "To test notifications:"
    echo "  1. Open $BASE_URL in your browser"
    echo "  2. Subscribe to topic '$TOPIC'"
    echo "  3. Run this script again: ./test-notification.sh $TOPIC"
else
    echo "   ✓ Notification sent to $SENT subscriber(s)"
fi

echo ""
echo "4. Sending plain text notification..."
curl -s -X POST "$BASE_URL/publish/$TOPIC" \
    -d "Plain text notification test" > /dev/null
echo "   ✓ Plain text notification sent"

echo ""
echo "Test complete!"
