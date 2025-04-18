# otel-config-lite

A simple and lightweight way to ingest and visualize telemetry data locally using Grafana without modifying your existing source files.

## Setup
<details>
<summary>Install Docker</summary>

- If you're on MacOS, you'll need to install Docker Desktop [here](https://docs.docker.com/desktop/setup/install/mac-install/)

- If you're on Windows, you'll need to install WSL and Docker Desktop [here](https://docs.docker.com/desktop/setup/install/windows-install/)

Try running any docker command (e.g `docker ps`) to make sure docker works properly.
>Note: If you run into the following error, it means you need to open the Docker Desktop app:
> ```
> Cannot connect to the Docker daemon at > unix:///var/run/docker.sock. Is the docker daemon running?
> ```
</details>

<details>
<summary>Setup `otel-config.go` in your project</summary>
Go to your project, and create a copy of the `otel-config.go` file in your project. By default the package is set to `package main`, but you'll need to modify this if you want to place it in a different package (e.g. to run tests).

Run the following to install the packages needed for the file:
```bash
go get "go.opentelemetry.io/contrib/exporters/autoexport" "go.opentelemetry.io/contrib/bridges/otelslog"
```
</details>

## Workflow

There are two parts to this workflow:
1) Run the Grafana stack on docker for ingesting and visualizing your telemetry data
2) Run your code (which has `otel-config.go` in it) to send your telemetry data to grafana.

To run the Grafana stack, run the following docker command in one terminal.
```bash
docker run -p 3000:3000 -p 4317:4317 -p 4318:4318 --rm -ti --name grafana grafana/otel-lgtm
```
<!-- docker run -p 3000:3000 -p 4317:4317 -p 4318:4318 --rm -ti --name grafana -v $(pwd)/otel-collector-config.yaml:/etc/otelcol-contrib/config.yaml -v $(pwd)/loki-config.yaml:/etc/loki/config.yaml grafana/otel-lgtm -->

> Note: If you run into the following error, it means you need to open the Docker Desktop app:
> ```
> Cannot connect to the Docker daemon at > unix:///var/run/docker.sock. Is the docker daemon running?
> ```

After around 30 seconds, you should see the program hanging after outputing some text.

<details>
<summary>Here's what the end of that text looks like:</summary>

```
Startup Time Summary:
---------------------
Grafana: 32 seconds
Loki: 3 seconds
Prometheus: 2 seconds
Tempo: 3 seconds
OpenTelemetry collector: 7 seconds
Total: 32 seconds
The OpenTelemetry collector and the Grafana LGTM stack are up and running. (created /tmp/ready)
Open ports:
 - 4317: OpenTelemetry GRPC endpoint
 - 4318: OpenTelemetry HTTP endpoint
 - 3000: Grafana. User: admin, password: admin
```
</details>

While this command is running, you can now open the grafana UI at [http://localhost:3000](http://localhost:3000) in your browser.

You can kill this command as normal using `ctrl + c`. When you do so, all of the data will be lost, and the browser will cease to run.
> There are ways around this, but this is nice because it's easy to "reset" and not have to worry about accumulating too much data and permanently persisting to disk.

Run your program and see your data show up in the UI.

For examples of how to instrument your program to generate telemetry data, see the [official OpenTelemetry Go docs](https://opentelemetry.io/docs/languages/go/getting-started/#add-custom-instrumentation).

Note: for logs to be exported, you need to use the `"log"` or `"slog"` package. If you currently, use the `"fmt"` package, you can replace all occurences with `log`. (e.g `log.Printf()`, `log.Println()`). These will automatically be exported to grafana (if it's running).
