# GOPETBIN
Pastebin alternative written in Go

# Installation

```sh
git clone https://github.com/swmh/gopetbin
cd gopetbin
make compose-dev-build && compose-dev-up
```

# Usage

## Create Paste

```sh
curl http://localhost:8080 --form 'content=lorem ipsum'
http://localhost:8080/7ZlwE4ADZe

curl http://localhost:8080 --form 'content=@file.txt' --form 'expire=10m' --form 'burn=3'
http://localhost:8080/4Gp3gCWeXl
```

## Get Paste

```sh
curl http://localhost:8080/7ZlwE4ADZe
lorem ipsum
```

## Expire time 
Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
