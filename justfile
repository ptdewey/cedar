all:
    @tailwindcss -i static/app.css -o public/style.css
    @go run main.go
    @cp ./static/darkearth-syntax.css ./public
    @cp ./static/bluesky-comments.js ./public

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
