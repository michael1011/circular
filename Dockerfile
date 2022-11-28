FROM golang:1.19.2-buster
ENTRYPOINT git clone https://github.com/michael1011/circular && cd circular && git checkout rebalance-broadcast && make
