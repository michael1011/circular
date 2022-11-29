FROM golang:1.19.2-buster
ENTRYPOINT git clone --depth=1 https://github.com/michael1011/circular && cd circular && make
