# balanzierer

## start regtest
```console
cd docker
chmod +x ./tests
./tests
```

## build go corelightning plugin for regtest instances
```console
cd docker
source docker-scripts.sh
balanzierer-build
```

## start the plugin on instance
```console
cd docker
source docker-scripts.sh
lightning-cli-sim 1 plugin start /circular/circular
```
