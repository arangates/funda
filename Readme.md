# funda-exporter
This is a scraper of Funda.nl website which presenting collected data as Prometheus series

Scraping initiated on the GET request call from metrics collector (Prometheus), so keep in mind that depends on the Funda URL it can take several minutes to finish.
So please consider to set relatively high `scrape_interval` and `scrape_timeout`.

Docker images could be found here: `vladikamira/funda-exporter:TAG_NAME`


parameters:
- `fakeUserAgent` (string)        - HTTP User-Agent to predend as a browserm otherwise Funda will block you
- `fundaSearchUrl` (string)       - Search URL with parameters like Price range, City, space size etc.. You feed your own URL here.
- `listenAddress` (string)        - Address to listen on
- `scrapeDelayMilliseconds` (Int) - Interval to add between scrapes to not overload Funda :)
- `postCodes` (string)            - List of Post Codes to search for (example: 1186 AB, 1185)

example to run with docker-compose:
```
version: "3.8"
services:
  funda-exporter:
    container_name: funda-exporter
    restart: always
    image: vladikamira/funda-exporter:v0.0.2
    command:
      - '-scrapeDelayMilliseconds=500'
      - '-fundaSearchUrl=https://www.funda.nl/koop/amstelveen,amsterdam/300000-440000/70+woonopp/2+slaapkamers/'
      - '-listenAddress=:2112'
    ports:
      - 2112:2112
```

and Prometheus scraping config could looks like that:
```
global:
  scrape_interval: 1m

scrape_configs:

  - job_name: 'funda-exporter'
    scrape_interval: 30m
    scrape_timeout: 29m
    static_configs:
    - targets: ['funda-exporter:2112']
```
