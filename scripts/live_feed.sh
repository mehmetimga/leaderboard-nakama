#!/bin/bash
# live_feed.sh - Continuously submits random scores to test real-time updates

API_URL="${API_URL:-http://localhost:8080}"
LEADERBOARD_ID="${LEADERBOARD_ID:-global_scores}"
INTERVAL="${INTERVAL:-2}"  # seconds between submissions

# Array of cool gamer names
ADJECTIVES=("Dark" "Swift" "Silent" "Fierce" "Mystic" "Cyber" "Neon" "Pixel" "Turbo" "Ultra" "Shadow" "Storm" "Thunder" "Blazing" "Frost" "Atomic" "Cosmic" "Epic" "Mega" "Super")
NOUNS=("Phantom" "Shadow" "Nova" "Blaze" "Storm" "Thunder" "Viper" "Ghost" "Raven" "Wolf" "Dragon" "Phoenix" "Titan" "Ninja" "Knight" "Hunter" "Warrior" "Legend" "Master" "Champion")

echo "🔴 LIVE FEED STARTED"
echo "📡 Submitting scores every ${INTERVAL}s to: $LEADERBOARD_ID"
echo "🛑 Press Ctrl+C to stop"
echo ""

count=0
while true; do
    count=$((count + 1))
    
    # Generate random name
    adj=${ADJECTIVES[$RANDOM % ${#ADJECTIVES[@]}]}
    noun=${NOUNS[$RANDOM % ${#NOUNS[@]}]}
    num=$((RANDOM % 999 + 1))
    username="${adj}${noun}${num}"
    user_id="live-$(cat /dev/urandom | LC_ALL=C tr -dc 'a-z0-9' | fold -w 8 | head -n 1)"
    
    # Generate random score (5000-100000 for more dramatic changes)
    score=$((RANDOM % 95000 + 5000))
    
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
    timestamp=$(date '+%H:%M:%S')
    
    if [ -n "$success" ]; then
        # Color code based on score
        if [ $score -gt 80000 ]; then
            echo "[$timestamp] 🏆 #$count HIGH SCORE! $username: $score pts"
        elif [ $score -gt 50000 ]; then
            echo "[$timestamp] ⭐ #$count $username: $score pts"
        else
            echo "[$timestamp] 📊 #$count $username: $score pts"
        fi
    else
        echo "[$timestamp] ❌ #$count Failed: $response"
    fi
    
    sleep $INTERVAL
done

