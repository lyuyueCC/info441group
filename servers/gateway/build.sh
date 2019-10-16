GOOS="linux"
go build
docker build -t atlantis/summary .
go clean