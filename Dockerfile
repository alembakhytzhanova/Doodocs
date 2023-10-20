
# Base Image
FROM golang:latest

WORKDIR /app
# Copy the source code into the container
COPY . .

# Build go files
RUN go build -o main ./cmd/main.go


# Set the entry point to run the go app
CMD ["./main"]

