# akari

## Usage
```sh
sntp -d ::1 # or 127.0.0.1, localhost
```

## Installation

### Run on local
```sh
go mod download
go run .
```

### Run on container
```sh
docker-compose up -d --build
```