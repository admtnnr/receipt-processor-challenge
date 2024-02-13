FROM golang:1.22

WORKDIR /usr/src/github.com/admtnnr/fetch
COPY . .

CMD ["go", "run", "github.com/admtnnr/fetch/cmd/fetch-api-server"]
