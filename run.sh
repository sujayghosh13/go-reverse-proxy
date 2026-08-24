#!/bin/bash
# Launch all backend go files in background
cd "$(dirname "$0")"

for f in backends/*.go; do
    name=$(basename "$f" .go)
    echo "Starting $name ($f)..."
    go run "$f" &
done

echo "All backends launched in background processes."
