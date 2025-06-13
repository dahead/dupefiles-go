#@/bin/sh

# linux
go build -o ../builds/linux/df ../main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o ../builds/windows/df.exe ../main.go