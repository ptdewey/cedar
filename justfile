all:
    @go run main.go
    @tailwindcss -i static/app.css -o public/style.css

run:
    @go run main.go

build:
    @go build

test:
    @go test ./... -cover -coverprofile=cover.out

style:
    @tailwindcss -i priv/app.css -o priv/build.css

clean:
    @rm -rf public
