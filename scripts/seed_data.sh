#!/bin/bash
# seed_data.sh - Seeds the leaderboard with random data

API_URL="${API_URL:-http://localhost:8080}"
LEADERBOARD_ID="${LEADERBOARD_ID:-global_scores}"
NUM_USERS="${NUM_USERS:-20}"

# Array of cool gamer names
ADJECTIVES=("Dark" "Swift" "Silent" "Fierce" "Mystic" "Cyber" "Neon" "Pixel" "Turbo" "Ultra" "Shadow" "Storm" "Thunder" "Blazing" "Frost")
NOUNS=("Phantom" "Shadow" "Nova" "Blaze" "Storm" "Thunder" "Viper" "Ghost" "Raven" "Wolf" "Dragon" "Phoenix" "Titan" "Ninja" "Knight")

echo "🎮 Seeding leaderboard: $LEADERBOARD_ID"
echo "📊 Creating $NUM_USERS users..."
echo ""

for i in $(seq 1 $NUM_USERS); do
    # Generate random name
    adj=${ADJECTIVES[$RANDOM % ${#ADJECTIVES[@]}]}
    noun=${NOUNS[$RANDOM % ${#NOUNS[@]}]}
    username="${adj}${noun}${i}"
    user_id="user-$(cat /dev/urandom | LC_ALL=C tr -dc 'a-z0-9' | fold -w 8 | head -n 1)"
    
    # Generate random score (1000-50000)
    score=$((RANDOM % 49000 + 1000))
    
    # Submit score
    response=$(curl -s -X POST "$API_URL/api/v1/scores" \
        -H "Content-Type: application/json" \
        -d "{
            \"leaderboard_id\": \"$LEADERBOARD_ID\",
            \"user_id\": \"$user_id\",
            \"username\": \"$username\",
            \"score\": $score
        }")
    
    success=$(echo "$response" | grep -o '"success":true' | head -1)
    
    if [ -n "$success" ]; then
        echo "✅ $username: $score pts"
    else
        echo "❌ Failed to submit score for $username"
        echo "   Response: $response"
    fi
    
    # Small delay to avoid overwhelming the server
    sleep 0.1
done

echo ""
echo "🏆 Seeding complete!"
echo "📈 View leaderboard at: $API_URL/api/v1/leaderboards/$LEADERBOARD_ID"

