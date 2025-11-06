all:
    @tailwindcss -i static/app.css -o static/main.css
    @go run main.go

run:
    @go run main.go

build:
    @go build

test:
    @go test ./... -cover -coverprofile=cover.out

style:
    @tailwindcss -i priv/app.css -o priv/build.css
