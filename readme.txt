
docker run -p 9090:9090 -v ./prometheus:/etc/prometheus/ prom/prometheus

docker run -d  -p 3000:3000 --name=grafana -e "GF_INSTALL_PLUGINS=grafana-clock-panel,grafana-simple-json-datasource" grafana/grafana