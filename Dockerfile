# Use a minimal Go base image
FROM golang:1.24-alpine

WORKDIR /app

# Install SQLite dependencies
RUN apk add --no-cache sqlite

# Copy files and build the app
COPY . .
RUN go build -o go-mud ./...

# Copy the entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Expose the game port
EXPOSE 4000

# Use the entrypoint script
ENTRYPOINT ["/entrypoint.sh"]
