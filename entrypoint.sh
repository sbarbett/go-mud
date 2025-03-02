#!/bin/sh

# Ensure the database file exists
if [ ! -f "/app/mud.db" ]; then
    echo "Creating mud.db..."
    touch /app/mud.db
    chmod 666 /app/mud.db
fi

# Start the MUD server
exec ./go-mud
