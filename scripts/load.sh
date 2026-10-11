#!/usr/bin/env sh
# Sends a mix of requests to the API so the dashboard has something to show.
# Usage: ./scripts/load.sh [base-url] [iterations]
BASE=${1:-http://localhost:8080}
N=${2:-200}
i=0
while [ "$i" -lt "$N" ]; do
  curl -s -o /dev/null "$BASE/cards"
  curl -s -o /dev/null "$BASE/cards/random"
  curl -s -o /dev/null "$BASE/cards/$((i % 5 + 1))"
  curl -s -o /dev/null -X POST "$BASE/cards" -H "Content-Type: application/json" \
    -d "{\"topic\":\"load\",\"question\":\"Q$i?\",\"answer\":\"A$i\"}"
  [ $((i % 10)) -eq 0 ] && curl -s -o /dev/null "$BASE/cards/not-a-number"
  i=$((i + 1))
done
echo "sent $((N * 4 + N / 10)) requests to $BASE"
