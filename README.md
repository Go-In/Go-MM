# Go-MM
### Real Time Mulware Attack Map based on Suricata

start server `development`
- configuration at src/config.json
- start redis server
```
go run main.go
```

start with `docker`
```
docker build -t go-mm .
docker run --rm -it -p [port]:[port] go-mm
```

