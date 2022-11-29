#!/bin/sh
export COMPOSE_PROJECT_NAME=balanzierer

balanzierer-build() {
    sudo rm -rf $(pwd)/circular-build
    docker build -t balanzierer ..
    docker run --rm --volume $(pwd)/circular-build:/go/circular balanzierer
    plugin-stop
    cp $(pwd)/circular-build/circular $(pwd)/circular/
    sudo chmod -R 777 $(pwd)/circular
    plugin-start
}

balanzierer-build-local() {
    cd ..
    make
    cd docker
    plugin-stop
    cp ../circular circular/
    plugin-start
}

plugin-start() {
    lightning-cli-sim 1 plugin start /circular/circular > /dev/null
}

plugin-stop() {
    lightning-cli-sim 1 plugin stop /circular/circular > /dev/null
}

bitcoin-cli-sim() {
  docker exec balanzierer-bitcoind-1 bitcoin-cli -rpcuser=balanzierer -rpcpassword=balanzierer -regtest $@
}

# args(i, cmd)
lightning-cli-sim() {
  i=$1
  shift # shift first argument so we can use $@
  docker exec balanzierer-clightning-$i-1 lightning-cli --network regtest $@
}

# args(i)
fund_clightning_node() {
  address=$(lightning-cli-sim $1 newaddr | jq -r .bech32)
  echo "funding: $address on clightning-node: $1"
  bitcoin-cli-sim -named sendtoaddress address=$address amount=30 fee_rate=100 > /dev/null
}

# args(i, j)
connect_clightning_node() {
  pubkey=$(lightning-cli-sim $2 getinfo | jq -r '.id')
  lightning-cli-sim $1 connect $pubkey@balanzierer-clightning-$2-1:9735 | jq -r '.id'
}

regtest-start(){
  regtest-stop
  docker compose up -d --remove-orphans
  regtest-init
}

regtest-start-log(){
  regtest-stop
  docker compose up --remove-orphans
  regtest-init
}

regtest-stop(){
  docker compose down --volumes
  # clean up lightning node data
  sudo rm -rf ./data/clightning-{1,2,3,4,5}
  # recreate lightning node data folders preventing permission errors
  mkdir ./data/clightning-{1,2,3,4,5}
}

regtest-restart(){
  regtest-stop
  regtest-start
}

bitcoin-init(){
  echo "init_bitcoin_wallet..."
  bitcoin-cli-sim createwallet balanzierer || bitcoin-cli-sim loadwallet balanzierer
  echo "mining 150 blocks..."
  bitcoin-cli-sim -generate 150 > /dev/null
}

regtest-init(){
  bitcoin-init
  lightning-sync
  lightning-init
}

lightning-sync(){
  wait-for-clightning-sync 1
  wait-for-clightning-sync 2
  wait-for-clightning-sync 3
  wait-for-clightning-sync 4
  wait-for-clightning-sync 5
}

lightning-init(){

  # create 5 UTXOs for each node
  for i in 0 1 2 3 4; do
    fund_clightning_node 1
    fund_clightning_node 2
    fund_clightning_node 3
    fund_clightning_node 4
    fund_clightning_node 5
  done

  echo "mining 10 blocks..."
  bitcoin-cli-sim -generate 10 > /dev/null

  echo "wait for 25s..."
  sleep 25 # else blockheight tests fail for cln

  lightning-sync

  channel_size=24000000 # 0.24 btc
  balance_size_msat=5000000000 # 0.05 btc

  # cln-1 -> cln-2
  peerid=$(connect_clightning_node 1 2)
  echo "open channel from cln-1 to cln-2"
  lightning-cli-sim 1 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  # cln-1 -> cln-4
  peerid=$(connect_clightning_node 1 4)
  echo "open channel from cln-1 to cln-4"
  lightning-cli-sim 1 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  # cln-1 -> cln-5
  peerid=$(connect_clightning_node 1 5)
  echo "open channel from cln-1 to cln-5"
  lightning-cli-sim 1 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  # cln-2 -> cln-3
  peerid=$(connect_clightning_node 2 3)
  echo "open channel from cln-2 to cln-3"
  lightning-cli-sim 2 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  # cln-2 -> cln-4
  peerid=$(connect_clightning_node 2 4)
  echo "open channel from cln-2 to cln-4"
  lightning-cli-sim 2 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  # cln-2 -> cln-5
  peerid=$(connect_clightning_node 2 5)
  echo "open channel from cln-2 to cln-5"
  lightning-cli-sim 2 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  # cln-3 -> cln-1
  peerid=$(connect_clightning_node 3 1)
  echo "open channel from cln-3 to cln-1"
  lightning-cli-sim 3 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  # cln-3 -> cln-4
  peerid=$(connect_clightning_node 3 4)
  echo "open channel from cln-3 to cln-4"
  lightning-cli-sim 3 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  # cln-3 -> cln-5
  peerid=$(connect_clightning_node 3 5)
  echo "open channel from cln-3 to cln-5"
  lightning-cli-sim 3 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  # cln-4 -> cln-5
  peerid=$(connect_clightning_node 4 5)
  echo "open channel from cln-4 to cln-5"
  lightning-cli-sim 4 fundchannel -k id=$peerid amount=$channel_size push_msat=$balance_size_msat > /dev/null

  bitcoin-cli-sim -generate 10 > /dev/null
  wait-for-clightning-channel 1
  wait-for-clightning-channel 2
  wait-for-clightning-channel 3
  wait-for-clightning-channel 4
  wait-for-clightning-channel 5
  lightning-sync

  echo "wait for 15s... warmup..."
  sleep 15

}

wait-for-clightning-channel(){
  while true; do
    pending=$(lightning-cli-sim $1 getinfo | jq -r '.num_pending_channels | length')
    echo "cln-$1 pendingchannels: $pending"
    if [[ "$pending" == "0" ]]; then
      if [[ "$(lightning-cli-sim $1 getinfo 2>&1 | jq -r '.warning_bitcoind_sync' 2> /dev/null)" == "null" ]]; then
        if [[ "$(lightning-cli-sim $1 getinfo 2>&1 | jq -r '.warning_lightningd_sync' 2> /dev/null)" == "null" ]]; then
          break
        fi
      fi
    fi
    sleep 1
  done
}

wait-for-clightning-sync(){
  while true; do
    if [[ ! "$(lightning-cli-sim $1 getinfo 2>&1 | jq -r '.id' 2> /dev/null)" == "null" ]]; then
      if [[ "$(lightning-cli-sim $1 getinfo 2>&1 | jq -r '.warning_bitcoind_sync' 2> /dev/null)" == "null" ]]; then
        if [[ "$(lightning-cli-sim $1 getinfo 2>&1 | jq -r '.warning_lightningd_sync' 2> /dev/null)" == "null" ]]; then
          echo "cln-$1 is synced!"
          break
        fi
      fi
    fi
    echo "waiting for cln-$1 to sync..."
    sleep 1
  done
}
