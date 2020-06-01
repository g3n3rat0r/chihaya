# Build docker image and run
```shell
go mod tidy # install missing deps
go mod vendor # save deps into vendor
docker build -t g3n3rat0r/chihaya:v2.0.0-rc.3 . # build new image
docker run -p 6880-6882:6880-6882 -v $PWD/config.yaml:/config.yaml:ro g3n3rat0r/chihaya:v2.0.0-rc.3 # run image
docker ps -a # get container id
docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 1b764780acc0 # get container IP
```

## Check tracker
1. Create a new torrent ad add url
```
http://172.17.0.2:34000/zbufb7u3it51228v2esffxnxibq3oroi/announce
```

2. Start the torrent and announce the tracker to see response
