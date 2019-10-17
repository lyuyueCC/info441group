GOOS="linux"
# Builds a Go executable for Linux
go build
# Builds the Docker container
docker build -t atlantis/summary .
# Deletes the Go executable for Linux
# so that it doesn't end up getting added to your repo
go clean